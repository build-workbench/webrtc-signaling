package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	redispubsub "signal/internal/store/redis"
	"signal/internal/observability"
	"signal/internal/room"
	"signal/internal/signaling"
)

type safeWS struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeWS) WriteJSON(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteJSON(v)
}

func (s *safeWS) WriteControl(messageType int, data []byte, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteControl(messageType, data, deadline)
}

// routeMessage sends a message to a specific peer or broadcasts to the room,
// with Redis fallback for multi-node deployments.
func (s *Server) routeMessage(roomID, peerID string, msg signaling.Envelope) {
	msg.RoomID = roomID
	msg.From = peerID
	msg.Ts = time.Now().UnixMilli()
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.To != "" {
		if err := s.rooms.SendTo(roomID, msg.To, msg); err != nil && s.bus != nil {
			_ = s.bus.PublishDirect(roomID, msg.To, msg)
		}
	} else {
		s.rooms.Broadcast(roomID, peerID, msg)
		if s.bus != nil {
			_ = s.bus.PublishBroadcast(roomID, peerID, msg)
		}
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		if ah := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(ah), "bearer ") {
			token = strings.TrimSpace(ah[7:])
		}
	}
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	claims, err := s.auth.ParseJoinToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	ws := &safeWS{Conn: conn}

	peerID := uuid.NewString()
	userID := claims.Subject
	role := claims.Role
	displayName := claims.DisplayName

	// limits & heartbeat
	conn.SetReadLimit(int64(s.cfg.Server.MaxMsgBytes))
	pongWait := time.Duration(s.cfg.Server.PongWaitSec) * time.Second
	pingInterval := time.Duration(s.cfg.Server.PingIntervalSec) * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	limiter := rate.NewLimiter(rate.Limit(s.cfg.Security.RateLimit.WSPerConnRPS), s.cfg.Security.RateLimit.WSBurst)

	observability.WSConnections.Inc()
	defer observability.WSConnections.Dec()

	// ping goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = ws.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			case <-done:
				return
			}
		}
	}()

	// read first message must be join
	var env signaling.Envelope
	if err := conn.ReadJSON(&env); err != nil {
		close(done)
		return
	}
	if env.Type != signaling.TypeJoin {
		_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2001, Message: "first message must be join"})})
		close(done)
		return
	}
	var jp signaling.JoinPayload
	_ = json.Unmarshal(env.Payload, &jp)
	roomID := jp.RoomID
	if roomID == "" {
		_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2001, Message: "roomId required"})})
		close(done)
		return
	}
	if claims.Rid != "" && claims.Rid != roomID {
		_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2003, Message: "forbidden room"})})
		close(done)
		return
	}
	if jp.DisplayName != "" {
		displayName = jp.DisplayName
	}
	if jp.Role != "" {
		role = jp.Role
	}

	p := &room.Participant{ID: peerID, UserID: userID, Role: role, DisplayName: displayName, Conn: ws, JoinedAt: time.Now()}
	peers, err := s.rooms.Join(roomID, p)
	if err != nil {
		_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2010, Message: err.Error()})})
		close(done)
		return
	}

	s.log.Info("peer joined", zap.String("peerID", peerID), zap.String("roomID", roomID), zap.String("userID", userID))

	// Subscribe Redis room channel if enabled
	if s.bus != nil {
		s.mu.Lock()
		if s.roomSubs[roomID] == 0 {
			if err := s.bus.SubscribeRoom(roomID, func(wm redispubsub.WireMessage) {
				switch wm.Kind {
				case redispubsub.KindDirect:
					_ = s.rooms.SendTo(wm.RoomID, wm.ToPeer, wm.Envelope)
				case redispubsub.KindBroadcast:
					s.rooms.Broadcast(wm.RoomID, wm.ExcludePeer, wm.Envelope)
				}
			}); err != nil {
				s.log.Warn("redis subscribe failed", zap.Error(err))
			}
		}
		s.roomSubs[roomID]++
		s.mu.Unlock()
	}

	// send joined + peers
	peersInfo := make([]map[string]any, 0, len(peers))
	for _, pp := range peers {
		peersInfo = append(peersInfo, map[string]any{"id": pp.ID, "role": pp.Role, "displayName": pp.DisplayName})
	}
	_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeJoined, RoomID: roomID, From: peerID, Payload: mustJSON(map[string]any{
		"self":       map[string]any{"id": peerID, "role": role, "displayName": displayName},
		"peers":      peersInfo,
		"iceServers": s.buildICEServers(),
	})})

	// notify others
	s.rooms.Broadcast(roomID, peerID, signaling.Envelope{Type: signaling.TypePeerJoin, RoomID: roomID, From: peerID, Payload: mustJSON(map[string]any{"id": peerID, "role": role, "displayName": displayName})})

	// read loop
	for {
		var msg signaling.Envelope
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		observability.MessagesInTotal.Inc()
		if !limiter.Allow() {
			_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2007, Message: "rate_limited"})})
			continue
		}
		switch msg.Type {
		case signaling.TypeOffer, signaling.TypeAnswer, signaling.TypeTrickle:
			if msg.To == "" {
				_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2001, Message: "missing to"})})
				continue
			}
			msg.RoomID = roomID
			msg.From = peerID
			msg.Ts = time.Now().UnixMilli()
			if msg.ID == "" {
				msg.ID = uuid.NewString()
			}
			if err := s.rooms.SendTo(roomID, msg.To, msg); err != nil && s.bus != nil {
				_ = s.bus.PublishDirect(roomID, msg.To, msg)
			}
		case signaling.TypeChat, signaling.TypeMute, signaling.TypeUnmute:
			s.routeMessage(roomID, peerID, msg)
		case signaling.TypeLeave:
			goto end
		default:
			_ = ws.WriteJSON(signaling.Envelope{Type: signaling.TypeError, Payload: mustJSON(signaling.ErrorPayload{Code: 2006, Message: "unsupported_type"})})
		}
	}

end:
	// cleanup
	close(done)
	if _, ok := s.rooms.Leave(roomID, peerID); ok {
		s.rooms.Broadcast(roomID, peerID, signaling.Envelope{Type: signaling.TypePeerLeave, RoomID: roomID, From: peerID, Payload: mustJSON(map[string]any{"id": peerID})})
		s.log.Info("peer left", zap.String("peerID", peerID), zap.String("roomID", roomID))
	}
	if s.bus != nil {
		s.mu.Lock()
		if s.roomSubs[roomID] > 0 {
			s.roomSubs[roomID]--
			if s.roomSubs[roomID] == 0 {
				_ = s.bus.UnsubscribeRoom(roomID)
				delete(s.roomSubs, roomID)
			}
		}
		s.mu.Unlock()
	}
}
