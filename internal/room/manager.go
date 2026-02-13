package room

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"signal/internal/observability"
	"signal/internal/signaling"
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
	Participants    map[string]*Participant
}

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	log   *zap.Logger
}

func NewManager(log *zap.Logger) *Manager {
	return &Manager{rooms: make(map[string]*Room), log: log}
}

func (m *Manager) CreateRoom(id string) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id == "" {
		id = uuid.NewString()
	}
	if _, ok := m.rooms[id]; ok {
		return m.rooms[id], nil
	}
	r := &Room{ID: id, Participants: map[string]*Participant{}}
	m.rooms[id] = r
	observability.RoomsGauge.Set(float64(len(m.rooms)))
	return r, nil
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
		r = &Room{ID: roomID, Participants: map[string]*Participant{}}
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
