package room

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/vibe-knight/aurora-signal/internal/observability"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── Signaling protocol types ────────────────────────────

type MessageType string

const (
	TypeJoin      MessageType = "join"
	TypeJoined    MessageType = "joined"
	TypeOffer     MessageType = "offer"
	TypeAnswer    MessageType = "answer"
	TypeTrickle   MessageType = "trickle"
	TypeLeave     MessageType = "leave"
	TypeChat      MessageType = "chat"
	TypeMute      MessageType = "mute"
	TypeUnmute    MessageType = "unmute"
	TypeError     MessageType = "error"
	TypePeerJoin  MessageType = "participant-joined"
	TypePeerLeave MessageType = "participant-left"
)

// Envelope is the wire format for every WebSocket message.
type Envelope struct {
	ID      string          `json:"id,omitempty"`
	Version string          `json:"version,omitempty"`
	Type    MessageType     `json:"type"`
	RoomID  string          `json:"roomId,omitempty"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Ts      int64           `json:"ts,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type JoinPayload struct {
	RoomID      string `json:"roomId"`
	DisplayName string `json:"displayName,omitempty"`
	Role        string `json:"role,omitempty"`
}

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ── Room / Participant ──────────────────────────────────

// SafeConn is the interface for a thread-safe WebSocket connection.
type SafeConn interface {
	WriteJSON(v any) error
}

// Participant represents a user in a room.
type Participant struct {
	ID          string
	UserID      string
	Role        string
	DisplayName string
	Conn        SafeConn
	JoinedAt    time.Time
}

// Room represents a signaling room.
type Room struct {
	ID              string
	MaxParticipants int
	CreatedAt       time.Time
	Participants    map[string]*Participant
}

// RoomSnapshot is an immutable view of a room returned to callers.
type RoomSnapshot struct {
	ID              string
	MaxParticipants int
}

var (
	ErrInvalidMaxParticipants = errors.New("max participants must be zero or positive")
	ErrInvalidParticipant     = errors.New("invalid participant")
	ErrRoomFull               = errors.New("room is full")
	ErrRoomNotFound           = errors.New("room not found")
	ErrPeerNotFound           = errors.New("peer not found")
)

// Manager manages rooms and participants.
type Manager struct {
	mu      sync.RWMutex
	rooms   map[string]*Room
	log     *zap.Logger
	stopCh  chan struct{}
	metrics observability.Metrics
}

// NewManager creates a new room manager.
func NewManager(log *zap.Logger, metrics observability.Metrics) *Manager {
	return &Manager{
		rooms:   make(map[string]*Room),
		log:     log,
		stopCh:  make(chan struct{}),
		metrics: metrics,
	}
}

func (m *Manager) StartCleanup(interval, emptyTTL time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.cleanupEmptyRooms(emptyTTL)
			case <-m.stopCh:
				return
			}
		}
	}()
}

func (m *Manager) Stop() { close(m.stopCh) }

func (m *Manager) cleanupEmptyRooms(ttl time.Duration) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, r := range m.rooms {
		if len(r.Participants) == 0 && now.Sub(r.CreatedAt) > ttl {
			delete(m.rooms, id)
			m.log.Debug("cleaned up empty room", zap.String("roomID", id))
		}
	}
	m.metrics.SetRooms(len(m.rooms))
}

// CreateRoom creates a room with the given id (auto-generated if empty) and
// maxParticipants cap (0 = unlimited). If a room with the same id already
// exists, it returns a snapshot of the existing room.
func (m *Manager) CreateRoom(id string, maxParticipants int) (RoomSnapshot, error) {
	if maxParticipants < 0 {
		return RoomSnapshot{}, ErrInvalidMaxParticipants
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		id = uuid.NewString()
	}
	if existing, ok := m.rooms[id]; ok {
		return RoomSnapshot{ID: existing.ID, MaxParticipants: existing.MaxParticipants}, nil
	}
	m.rooms[id] = &Room{
		ID:              id,
		MaxParticipants: maxParticipants,
		CreatedAt:       time.Now(),
		Participants:    map[string]*Participant{},
	}
	m.metrics.SetRooms(len(m.rooms))
	return RoomSnapshot{ID: id, MaxParticipants: maxParticipants}, nil
}

// RoomInfo returns the room id and current participant count.
func (m *Manager) RoomInfo(id string) (roomID string, participants int, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[id]
	if !ok {
		return "", 0, false
	}
	return r.ID, len(r.Participants), true
}

func (m *Manager) Join(roomID string, p *Participant) ([]*Participant, error) {
	if p == nil || p.Conn == nil {
		return nil, ErrInvalidParticipant
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		r = &Room{ID: roomID, CreatedAt: time.Now(), Participants: map[string]*Participant{}}
		m.rooms[roomID] = r
		m.metrics.SetRooms(len(m.rooms))
	}
	if r.MaxParticipants > 0 && len(r.Participants) >= r.MaxParticipants {
		return nil, ErrRoomFull
	}
	peers := make([]*Participant, 0, len(r.Participants))
	for _, v := range r.Participants {
		peers = append(peers, v)
	}
	r.Participants[p.ID] = p
	m.metrics.IncParticipants()
	return peers, nil
}

func (m *Manager) Leave(roomID, peerID string) (*Participant, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, false
	}
	p, exists := r.Participants[peerID]
	if !exists {
		return nil, false
	}
	delete(r.Participants, peerID)
	m.metrics.DecParticipants()
	if len(r.Participants) == 0 {
		delete(m.rooms, roomID)
		m.metrics.SetRooms(len(m.rooms))
	}
	return p, true
}

func (m *Manager) SendTo(roomID, toPeerID string, env Envelope) error {
	m.mu.RLock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.RUnlock()
		return ErrRoomNotFound
	}
	p, ok := r.Participants[toPeerID]
	if !ok {
		m.mu.RUnlock()
		return ErrPeerNotFound
	}
	conn := p.Conn
	m.mu.RUnlock()

	if err := conn.WriteJSON(env); err != nil {
		return err
	}
	m.metrics.IncMessagesOut()
	return nil
}

func (m *Manager) Broadcast(roomID, excludePeerID string, env Envelope) {
	m.mu.RLock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.RUnlock()
		return
	}
	targets := make([]SafeConn, 0, len(r.Participants))
	for id, p := range r.Participants {
		if id == excludePeerID {
			continue
		}
		targets = append(targets, p.Conn)
	}
	m.mu.RUnlock()

	for _, conn := range targets {
		if err := conn.WriteJSON(env); err == nil {
			m.metrics.IncMessagesOut()
		}
	}
}
