package room

import (
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"signal/internal/signaling"
)

type mockConn struct{ n int64 }

func (m *mockConn) WriteJSON(v any) error { atomic.AddInt64(&m.n, 1); return nil }

func TestRoomLifecycle(t *testing.T) {
	log, _ := zap.NewDevelopment()
	mgr := NewManager(log)
	r, err := mgr.CreateRoom("")
	if err != nil { t.Fatal(err) }
	if r.ID == "" { t.Fatal("room id empty") }

	p1 := &Participant{ID: "p1", UserID: "u1", Role: "speaker", DisplayName: "A", Conn: &mockConn{}, JoinedAt: time.Now()}
	peers, err := mgr.Join(r.ID, p1)
	if err != nil { t.Fatal(err) }
	if len(peers) != 0 { t.Fatalf("expected 0 peers, got %d", len(peers)) }

	p2 := &Participant{ID: "p2", UserID: "u2", Role: "speaker", DisplayName: "B", Conn: &mockConn{}, JoinedAt: time.Now()}
	peers, err = mgr.Join(r.ID, p2)
	if err != nil { t.Fatal(err) }
	if len(peers) != 1 { t.Fatalf("expected 1 peer before join, got %d", len(peers)) }

	// send direct
	if err := mgr.SendTo(r.ID, "p2", signaling.Envelope{Type: signaling.TypeChat}); err != nil { t.Fatal(err) }
	// broadcast
	mgr.Broadcast(r.ID, "p2", signaling.Envelope{Type: signaling.TypeChat})

	if _, ok := mgr.Leave(r.ID, "p1"); !ok { t.Fatal("leave p1 failed") }
	if _, ok := mgr.Leave(r.ID, "p2"); !ok { t.Fatal("leave p2 failed") }
	if _, ok := mgr.GetRoom(r.ID); ok { t.Fatal("room should be deleted when empty") }
}
