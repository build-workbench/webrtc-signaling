package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LessUp/aurora-signal/internal/permission"
	"github.com/LessUp/aurora-signal/internal/room"
	"github.com/LessUp/aurora-signal/internal/signaling"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type safeWS struct {
	Conn         *websocket.Conn
	mu           sync.Mutex
	writeTimeout time.Duration
}

func (s *safeWS) WriteJSON(v any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writeTimeout > 0 {
		if err := s.Conn.SetWriteDeadline(time.Now().Add(s.writeTimeout)); err != nil {
			return err
		}
		defer func() { _ = s.Conn.SetWriteDeadline(time.Time{}) }()
	}
	return s.Conn.WriteJSON(v)
}

func (s *safeWS) WriteControl(messageType int, data []byte, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Conn.WriteControl(messageType, data, deadline)
}

type wsSession struct {
	srv         *Server
	ws          *safeWS
	conn        *websocket.Conn
	peerID      string
	userID      string
	role        permission.Role
	displayName string
	roomID      string
	limiter     *rate.Limiter
	done        chan struct{}
}

func (s *Server) newSession(conn *websocket.Conn, userID, role, displayName string) *wsSession {
	normalizedRole := s.policy.Normalize(role)
	if normalizedRole == "" {
		normalizedRole = permission.RoleSpeaker
	}
	return &wsSession{
		srv:         s,
		ws:          &safeWS{Conn: conn, writeTimeout: 5 * time.Second},
		conn:        conn,
		peerID:      uuid.NewString(),
		userID:      userID,
		role:        normalizedRole,
		displayName: strings.TrimSpace(displayName),
		limiter:     rate.NewLimiter(rate.Limit(s.cfg.Security.RateLimit.WSPerConnRPS), s.cfg.Security.RateLimit.WSBurst),
		done:        make(chan struct{}),
	}
}

func (sess *wsSession) sendError(code int, message string) {
	sess.srv.metrics.IncError(code)
	_ = sess.ws.WriteJSON(signaling.Envelope{
		Type:    signaling.TypeError,
		Payload: mustJSON(signaling.ErrorPayload{Code: code, Message: message}),
	})
}

func (sess *wsSession) canSignalMedia() bool {
	return sess.srv.policy.CanSignalMedia(sess.role)
}

func (sess *wsSession) canModerateOthers() bool {
	return sess.srv.policy.CanModerateOthers(sess.role)
}

func (sess *wsSession) startPing(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = sess.ws.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			case <-sess.done:
				return
			}
		}
	}()
}

func (sess *wsSession) joinRoom(tokenRoomID string) ([]*room.Participant, bool) {
	var env signaling.Envelope
	if err := sess.conn.ReadJSON(&env); err != nil {
		return nil, false
	}
	if env.Type != signaling.TypeJoin {
		sess.sendError(2001, "first message must be join")
		return nil, false
	}
	sess.srv.metrics.IncMessagesIn()

	var jp signaling.JoinPayload
	if err := json.Unmarshal(env.Payload, &jp); err != nil {
		sess.sendError(2001, "invalid join payload")
		return nil, false
	}
	if jp.RoomID == "" {
		sess.sendError(2001, "roomId required")
		return nil, false
	}
	if tokenRoomID != "" && tokenRoomID != jp.RoomID {
		sess.sendError(2003, "forbidden room")
		return nil, false
	}
	if jp.Role != "" && sess.srv.policy.Normalize(jp.Role) == "" {
		sess.sendError(2001, "invalid role")
		return nil, false
	}

	sess.roomID = jp.RoomID
	if strings.TrimSpace(jp.DisplayName) != "" {
		sess.displayName = strings.TrimSpace(jp.DisplayName)
	}
	// Role was already normalized in newSession

	p := &room.Participant{
		ID:          sess.peerID,
		UserID:      sess.userID,
		Role:        string(sess.role),
		DisplayName: sess.displayName,
		Conn:        sess.ws,
		JoinedAt:    time.Now(),
	}
	peers, err := sess.srv.rooms.Join(sess.roomID, p)
	if err != nil {
		sess.sendError(2010, err.Error())
		return nil, false
	}
	return peers, true
}

func (sess *wsSession) notifyJoined(peers []*room.Participant) {
	peersInfo := make([]map[string]any, 0, len(peers))
	for _, pp := range peers {
		peersInfo = append(peersInfo, map[string]any{
			"id": pp.ID, "role": pp.Role, "displayName": pp.DisplayName,
		})
	}
	_ = sess.ws.WriteJSON(signaling.Envelope{
		Type:   signaling.TypeJoined,
		RoomID: sess.roomID,
		From:   sess.peerID,
		Payload: mustJSON(map[string]any{
			"self":       map[string]any{"id": sess.peerID, "role": string(sess.role), "displayName": sess.displayName},
			"peers":      peersInfo,
			"iceServers": sess.srv.buildICEServers(),
		}),
	})

	sess.srv.rooms.Broadcast(sess.roomID, sess.peerID, signaling.Envelope{
		Type:   signaling.TypePeerJoin,
		RoomID: sess.roomID,
		From:   sess.peerID,
		Payload: mustJSON(map[string]any{
			"id": sess.peerID, "role": string(sess.role), "displayName": sess.displayName,
		}),
	})
}

func (sess *wsSession) readLoop() {
	for {
		var msg signaling.Envelope
		if err := sess.conn.ReadJSON(&msg); err != nil {
			return
		}
		sess.srv.metrics.IncMessagesIn()

		if !sess.limiter.Allow() {
			sess.sendError(2007, "rate_limited")
			continue
		}

		switch msg.Type {
		case signaling.TypeOffer, signaling.TypeAnswer, signaling.TypeTrickle:
			if !sess.canSignalMedia() {
				sess.sendError(2003, "viewers cannot send media signaling")
				continue
			}
			if msg.To == "" {
				sess.sendError(2001, "missing to")
				continue
			}
			sess.srv.router.Route(sess.roomID, sess.peerID, msg)

		case signaling.TypeChat:
			sess.srv.router.Route(sess.roomID, sess.peerID, msg)

		case signaling.TypeMute, signaling.TypeUnmute:
			if msg.To != "" && msg.To != sess.peerID && !sess.canModerateOthers() {
				sess.sendError(2003, "only moderators can mute/unmute others")
				continue
			}
			sess.srv.router.Route(sess.roomID, sess.peerID, msg)

		case signaling.TypeLeave:
			return

		default:
			sess.sendError(2006, "unsupported_type")
		}
	}
}

func (sess *wsSession) cleanup() {
	_ = sess.ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
		time.Now().Add(3*time.Second))

	close(sess.done)

	if sess.roomID == "" {
		return
	}

	if _, ok := sess.srv.rooms.Leave(sess.roomID, sess.peerID); ok {
		sess.srv.rooms.Broadcast(sess.roomID, sess.peerID, signaling.Envelope{
			Type:    signaling.TypePeerLeave,
			RoomID:  sess.roomID,
			From:    sess.peerID,
			Payload: mustJSON(map[string]any{"id": sess.peerID}),
		})
		sess.srv.log.Info("peer left", zap.String("peerID", sess.peerID), zap.String("roomID", sess.roomID))
	}

	sess.srv.unsubscribeRoomIfLast(sess.roomID)
}

func (s *Server) subscribeRoomRedis(roomID string) {
	if s.bus == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roomSubs[roomID] == 0 {
		if err := s.bus.SubscribeRoom(roomID, s.router.HandleWireMessage); err != nil {
			s.log.Warn("redis subscribe failed", zap.Error(err), zap.String("roomID", roomID))
			return
		}
	}
	s.roomSubs[roomID]++
}

func (s *Server) unsubscribeRoomIfLast(roomID string) {
	if s.bus == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roomSubs[roomID] > 0 {
		s.roomSubs[roomID]--
		if s.roomSubs[roomID] == 0 {
			_ = s.bus.UnsubscribeRoom(roomID)
			delete(s.roomSubs, roomID)
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
	s.trackConn(conn)
	defer s.untrackConn(conn)

	sess := s.newSession(conn, claims.Subject, claims.Role, claims.DisplayName)
	defer sess.cleanup()

	pongWait := time.Duration(s.cfg.Server.PongWaitSec) * time.Second
	pingInterval := time.Duration(s.cfg.Server.PingIntervalSec) * time.Second
	conn.SetReadLimit(int64(s.cfg.Server.MaxMsgBytes))
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	s.metrics.IncConnections()
	defer s.metrics.DecConnections()

	sess.startPing(pingInterval)

	peers, ok := sess.joinRoom(claims.Rid)
	if !ok {
		return
	}
	s.log.Info("peer joined",
		zap.String("peerID", sess.peerID),
		zap.String("roomID", sess.roomID),
		zap.String("userID", sess.userID),
	)
	s.subscribeRoomRedis(sess.roomID)
	sess.notifyJoined(peers)
	sess.readLoop()
}
