package room

import (
	"errors"
	"sync"
	"time"

	"github.com/LessUp/aurora-signal/internal/observability"
	"github.com/LessUp/aurora-signal/internal/signaling"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
	if metrics == nil {
		metrics = observability.NewNoopMetrics()
	}
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

func (m *Manager) Stop() {
	close(m.stopCh)
}

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

func (m *Manager) CreateRoom(id string, maxParticipants ...int) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		id = uuid.NewString()
	}
	if existing, ok := m.rooms[id]; ok {
		return cloneRoom(existing), nil
	}
	r := &Room{ID: id, CreatedAt: time.Now(), Participants: map[string]*Participant{}}
	if len(maxParticipants) > 0 {
		if maxParticipants[0] < 0 {
			return nil, ErrInvalidMaxParticipants
		}
		if maxParticipants[0] > 0 {
			r.MaxParticipants = maxParticipants[0]
		}
	}
	m.rooms[id] = r
	m.metrics.SetRooms(len(m.rooms))
	return cloneRoom(r), nil
}

func (r *Room) ParticipantCount() int {
	return len(r.Participants)
}

func (m *Manager) GetRoom(id string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[id]
	if !ok {
		return nil, false
	}
	return cloneRoom(r), true
}

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
		if roomID == "" {
			roomID = uuid.NewString()
		}
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

func (m *Manager) ListPeers(roomID string) []*Participant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil
	}
	res := make([]*Participant, 0, len(r.Participants))
	for _, p := range r.Participants {
		res = append(res, p)
	}
	return res
}

func (m *Manager) SendTo(roomID, toPeerID string, env signaling.Envelope) error {
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

func (m *Manager) Broadcast(roomID, excludePeerID string, env signaling.Envelope) {
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

func (m *Manager) RoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}

func cloneRoom(r *Room) *Room {
	clone := &Room{
		ID:              r.ID,
		MaxParticipants: r.MaxParticipants,
		CreatedAt:       r.CreatedAt,
		Participants:    make(map[string]*Participant, len(r.Participants)),
	}
	for id, p := range r.Participants {
		clone.Participants[id] = p
	}
	return clone
}
