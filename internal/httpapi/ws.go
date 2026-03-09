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
	redispubsub "github.com/LessUp/aurora-signal/internal/store/redis"
	"github.com/LessUp/aurora-signal/internal/observability"
	"github.com/LessUp/aurora-signal/internal/room"
	"github.com/LessUp/aurora-signal/internal/signaling"
)

// ---------------------------------------------------------------------------
// safeWS wraps a *websocket.Conn with a mutex so concurrent goroutines (ping
// ticker, read loop, broadcast) can safely write.
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// wsSession holds the full state of a single WebSocket connection. Extracting
// this from Server keeps the handler readable and testable.
// ---------------------------------------------------------------------------

type wsSession struct {
	srv         *Server
	ws          *safeWS
	conn        *websocket.Conn
	peerID      string
	userID      string
	role        string
	displayName string
	roomID      string
	limiter     *rate.Limiter
	done        chan struct{} // closed to stop ping goroutine
}

func (s *Server) newSession(conn *websocket.Conn, userID, role, displayName string) *wsSession {
	return &wsSession{
		srv:         s,
		ws:          &safeWS{Conn: conn},
		conn:        conn,
		peerID:      uuid.NewString(),
		userID:      userID,
		role:        role,
		displayName: displayName,
		limiter:     rate.NewLimiter(rate.Limit(s.cfg.Security.RateLimit.WSPerConnRPS), s.cfg.Security.RateLimit.WSBurst),
		done:        make(chan struct{}),
	}
}

// sendError is a helper to write a typed error envelope.
func (sess *wsSession) sendError(code int, message string) {
	_ = sess.ws.WriteJSON(signaling.Envelope{
		Type:    signaling.TypeError,
		Payload: mustJSON(signaling.ErrorPayload{Code: code, Message: message}),
	})
}

// startPing launches the heartbeat goroutine. It returns immediately.
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

// joinRoom reads the mandatory first "join" message, validates it against the
// JWT claims, and adds the participant to the room. Returns the list of
// existing peers, or an error string if the join failed (error already sent to
// the client in that case).
func (sess *wsSession) joinRoom(tokenRoomID string) ([]*room.Participant, bool) {
	var env signaling.Envelope
	if err := sess.conn.ReadJSON(&env); err != nil {
		return nil, false
	}
	if env.Type != signaling.TypeJoin {
		sess.sendError(2001, "first message must be join")
		return nil, false
	}

	var jp signaling.JoinPayload
	_ = json.Unmarshal(env.Payload, &jp)

	if jp.RoomID == "" {
		sess.sendError(2001, "roomId required")
		return nil, false
	}
	if tokenRoomID != "" && tokenRoomID != jp.RoomID {
		sess.sendError(2003, "forbidden room")
		return nil, false
	}
	sess.roomID = jp.RoomID
	if jp.DisplayName != "" {
		sess.displayName = jp.DisplayName
	}
	if jp.Role != "" {
		sess.role = jp.Role
	}

	p := &room.Participant{
		ID:          sess.peerID,
		UserID:      sess.userID,
		Role:        sess.role,
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

// notifyJoined sends the "joined" envelope to this peer and broadcasts
// "participant-joined" to all others.
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
			"self":       map[string]any{"id": sess.peerID, "role": sess.role, "displayName": sess.displayName},
			"peers":      peersInfo,
			"iceServers": sess.srv.buildICEServers(),
		}),
	})

	sess.srv.rooms.Broadcast(sess.roomID, sess.peerID, signaling.Envelope{
		Type:   signaling.TypePeerJoin,
		RoomID: sess.roomID,
		From:   sess.peerID,
		Payload: mustJSON(map[string]any{
			"id": sess.peerID, "role": sess.role, "displayName": sess.displayName,
		}),
	})
}

// readLoop processes messages until the client disconnects or sends "leave".
func (sess *wsSession) readLoop() {
	for {
		var msg signaling.Envelope
		if err := sess.conn.ReadJSON(&msg); err != nil {
			return // connection closed or read error
		}
		observability.MessagesInTotal.Inc()

		if !sess.limiter.Allow() {
			sess.sendError(2007, "rate_limited")
			continue
		}

		switch msg.Type {
		case signaling.TypeOffer, signaling.TypeAnswer, signaling.TypeTrickle:
			if sess.role == "viewer" {
				sess.sendError(2003, "viewers cannot send media signaling")
				continue
			}
			if msg.To == "" {
				sess.sendError(2001, "missing to")
				continue
			}
			sess.srv.routeMessage(sess.roomID, sess.peerID, msg)

		case signaling.TypeChat:
			sess.srv.routeMessage(sess.roomID, sess.peerID, msg)

		case signaling.TypeMute, signaling.TypeUnmute:
			// mute/unmute targeting another peer requires moderator role
			if msg.To != "" && msg.To != sess.peerID && sess.role != "moderator" {
				sess.sendError(2003, "only moderators can mute/unmute others")
				continue
			}
			sess.srv.routeMessage(sess.roomID, sess.peerID, msg)

		case signaling.TypeLeave:
			return

		default:
			sess.sendError(2006, "unsupported_type")
		}
	}
}

// cleanup sends a close frame, removes the peer from the room, broadcasts
// departure, and tears down the Redis subscription refcount.
func (sess *wsSession) cleanup() {
	_ = sess.ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
		time.Now().Add(3*time.Second))

	close(sess.done) // stop ping goroutine

	if sess.roomID == "" {
		return // never joined a room
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

// ---------------------------------------------------------------------------
// Server-level helpers
// ---------------------------------------------------------------------------

// routeMessage stamps envelope metadata and sends to a specific peer or
// broadcasts to the room, with Redis fallback for multi-node deployments.
func (s *Server) routeMessage(roomID, peerID string, msg signaling.Envelope) {
	now := time.Now()
	msg.Version = "v1"
	msg.RoomID = roomID
	msg.From = peerID
	msg.Ts = now.UnixMilli()
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	defer func() {
		observability.MessageLatency.Observe(time.Since(now).Seconds())
	}()
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

// subscribeRoomRedis increments the per-room subscriber refcount and
// subscribes the Redis channel if this is the first local participant.
func (s *Server) subscribeRoomRedis(roomID string) {
	if s.bus == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
}

// unsubscribeRoomIfLast decrements the refcount and unsubscribes when it hits 0.
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

// ---------------------------------------------------------------------------
// handleWS is the HTTP handler for /ws/v1. It authenticates the token,
// upgrades to WebSocket, and runs the session lifecycle.
// ---------------------------------------------------------------------------

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// --- authenticate ---
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

	// --- upgrade ---
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	s.trackConn(conn)
	defer s.untrackConn(conn)

	sess := s.newSession(conn, claims.Subject, claims.Role, claims.DisplayName)
	defer sess.cleanup()

	// --- configure connection ---
	pongWait := time.Duration(s.cfg.Server.PongWaitSec) * time.Second
	pingInterval := time.Duration(s.cfg.Server.PingIntervalSec) * time.Second
	conn.SetReadLimit(int64(s.cfg.Server.MaxMsgBytes))
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	observability.WSConnections.Inc()
	defer observability.WSConnections.Dec()

	sess.startPing(pingInterval)

	// --- join ---
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

	// --- read loop (blocks until disconnect or leave) ---
	sess.readLoop()
}
