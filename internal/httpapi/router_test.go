package httpapi

import (
	"errors"
	"testing"

	"github.com/AICL-Lab/aurora-signal/internal/observability"
	"github.com/AICL-Lab/aurora-signal/internal/room"
	"go.uber.org/zap"
)

type stubSender struct {
	sendErr error
}

func (s *stubSender) SendTo(string, string, room.Envelope) error { return s.sendErr }
func (s *stubSender) Broadcast(string, string, room.Envelope)    {}

type stubBus struct {
	directCalls int
	lastRoomID  string
	lastToPeer  string
	lastEnv     room.Envelope
}

func (b *stubBus) PublishDirect(roomID, toPeer string, env room.Envelope) error {
	b.directCalls++
	b.lastRoomID = roomID
	b.lastToPeer = toPeer
	b.lastEnv = env
	return nil
}
func (b *stubBus) PublishBroadcast(string, string, room.Envelope) error { return nil }
func (b *stubBus) SubscribeRoom(string, func(WireMessage)) error         { return nil }
func (b *stubBus) UnsubscribeRoom(string) error                          { return nil }
func (b *stubBus) Close() error                                          { return nil }

func TestRouteDirectFallsBackToBusWhenPeerMissing(t *testing.T) {
	sender := &stubSender{sendErr: room.ErrPeerNotFound}
	bus := &stubBus{}
	rt := NewRouter(sender, bus, zap.NewNop(), observability.NewNoopMetrics())

	rt.Route("room-1", "peer-a", room.Envelope{Type: room.TypeOffer, To: "peer-b"})

	if bus.directCalls != 1 {
		t.Fatalf("expected one bus fallback, got %d", bus.directCalls)
	}
	if bus.lastRoomID != "room-1" || bus.lastToPeer != "peer-b" {
		t.Fatalf("unexpected bus target: room=%q to=%q", bus.lastRoomID, bus.lastToPeer)
	}
	if bus.lastEnv.Version != "v1" || bus.lastEnv.RoomID != "room-1" || bus.lastEnv.From != "peer-a" || bus.lastEnv.Ts == 0 {
		t.Fatalf("expected routed envelope metadata to be populated, got %+v", bus.lastEnv)
	}
}

func TestRouteDirectDoesNotFallbackOnLocalWriteError(t *testing.T) {
	sender := &stubSender{sendErr: errors.New("broken pipe")}
	bus := &stubBus{}
	rt := NewRouter(sender, bus, zap.NewNop(), observability.NewNoopMetrics())

	rt.Route("room-1", "peer-a", room.Envelope{Type: room.TypeOffer, To: "peer-b"})

	if bus.directCalls != 0 {
		t.Fatalf("expected no bus fallback for local delivery errors, got %d", bus.directCalls)
	}
}
