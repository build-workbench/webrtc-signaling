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

type SafeConn interface {
	WriteJSON(v any) error
}

type Participant struct {
	ID          string
	UserID      string
	Role        string
	DisplayName string
	Conn        SafeConn
	JoinedAt    time.Time
}

type Room struct {
	ID              string
	MaxParticipants int
	CreatedAt       time.Time
	Participants    map[string]*Participant
}

type Manager struct {
	mu     sync.RWMutex
	rooms  map[string]*Room
	log    *zap.Logger
	stopCh chan struct{}
}

func NewManager(log *zap.Logger) *Manager {
	return &Manager{rooms: make(map[string]*Room), log: log, stopCh: make(chan struct{})}
}

// StartCleanup runs a background goroutine that removes empty rooms
// older than the given TTL. Call Stop() to terminate.
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
	observability.RoomsGauge.Set(float64(len(m.rooms)))
}

func (m *Manager) CreateRoom(id string, maxParticipants ...int) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		id = uuid.NewString()
	}
	if existing, ok := m.rooms[id]; ok {
		return existing, nil
	}
	r := &Room{ID: id, CreatedAt: time.Now(), Participants: map[string]*Participant{}}
	if len(maxParticipants) > 0 && maxParticipants[0] > 0 {
		r.MaxParticipants = maxParticipants[0]
	}
	m.rooms[id] = r
	observability.RoomsGauge.Set(float64(len(m.rooms)))
	return r, nil
}

// ParticipantCount returns the number of participants in the room.
// Safe to call concurrently as a snapshot.
func (r *Room) ParticipantCount() int {
	return len(r.Participants)
}

func (m *Manager) GetRoom(id string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[id]
	return r, ok
}

func (m *Manager) Join(roomID string, p *Participant) ([]*Participant, error) {
	if p == nil || p.Conn == nil {
		return nil, errors.New("invalid participant")
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
		observability.RoomsGauge.Set(float64(len(m.rooms)))
	}
	if r.MaxParticipants > 0 && len(r.Participants) >= r.MaxParticipants {
		return nil, errors.New("room is full")
	}
	// snapshot peers before adding
	peers := make([]*Participant, 0, len(r.Participants))
	for _, v := range r.Participants {
		peers = append(peers, v)
	}
	r.Participants[p.ID] = p
	observability.ParticipantsGauge.Inc()
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
	observability.ParticipantsGauge.Dec()
	if len(r.Participants) == 0 {
		delete(m.rooms, roomID)
		observability.RoomsGauge.Set(float64(len(m.rooms)))
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
		return errors.New("room not found")
	}
	p, ok := r.Participants[toPeerID]
	if !ok {
		m.mu.RUnlock()
		return errors.New("peer not found")
	}
	conn := p.Conn
	m.mu.RUnlock()

	if err := conn.WriteJSON(env); err != nil {
		return err
	}
	observability.MessagesOutTotal.Inc()
	return nil
}

func (m *Manager) Broadcast(roomID, excludePeerID string, env signaling.Envelope) {
	m.mu.RLock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.RUnlock()
		return
	}
	// snapshot connections to release lock before network I/O
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
			observability.MessagesOutTotal.Inc()
		}
	}
}

func (m *Manager) RoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}
