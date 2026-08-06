package httpapi

import (
	"errors"
	"time"

	"github.com/AICL-Lab/aurora-signal/internal/observability"
	"github.com/AICL-Lab/aurora-signal/internal/room"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RoomSender defines the interface for sending messages within a room.
type RoomSender interface {
	SendTo(roomID, toPeerID string, env room.Envelope) error
	Broadcast(roomID, excludePeerID string, env room.Envelope)
}

// Bus defines the interface for cross-node message distribution.
type Bus interface {
	PublishDirect(roomID, toPeer string, env room.Envelope) error
	PublishBroadcast(roomID, excludePeer string, env room.Envelope) error
	SubscribeRoom(roomID string, handler func(WireMessage)) error
	UnsubscribeRoom(roomID string) error
	Close() error
}

// Router handles message routing between local and cross-node destinations.
type Router struct {
	sender  RoomSender
	bus     Bus
	log     *zap.Logger
	metrics observability.Metrics
}

func NewRouter(sender RoomSender, bus Bus, log *zap.Logger, metrics observability.Metrics) *Router {
	return &Router{sender: sender, bus: bus, log: log, metrics: metrics}
}

// Route delivers a message to its destination(s).
// If msg.To is set, sends directly to that peer (locally first, then via bus
// if the peer is not on this node). Otherwise broadcasts to the whole room.
func (r *Router) Route(roomID, peerID string, msg room.Envelope) {
	now := time.Now()
	msg.Version = "v1"
	msg.RoomID = roomID
	msg.From = peerID
	msg.Ts = now.UnixMilli()
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	defer func() {
		r.metrics.RecordLatency(time.Since(now).Seconds())
	}()

	if msg.To != "" {
		r.routeDirect(roomID, msg)
		return
	}
	r.routeBroadcast(roomID, peerID, msg)
}

func (r *Router) routeDirect(roomID string, msg room.Envelope) {
	if err := r.sender.SendTo(roomID, msg.To, msg); err != nil {
		if shouldFallbackToBus(err) && r.bus != nil {
			if pubErr := r.bus.PublishDirect(roomID, msg.To, msg); pubErr != nil {
				r.log.Warn("redis direct publish failed",
					zap.Error(pubErr),
					zap.String("roomID", roomID),
					zap.String("toPeer", msg.To))
			}
			return
		}
		r.log.Warn("local direct delivery failed",
			zap.Error(err),
			zap.String("roomID", roomID),
			zap.String("toPeer", msg.To))
	}
}

func (r *Router) routeBroadcast(roomID, peerID string, msg room.Envelope) {
	r.sender.Broadcast(roomID, peerID, msg)
	if r.bus != nil {
		if err := r.bus.PublishBroadcast(roomID, peerID, msg); err != nil {
			r.log.Warn("redis broadcast publish failed",
				zap.Error(err),
				zap.String("roomID", roomID))
		}
	}
}

// HandleWireMessage processes a message received from the bus.
func (r *Router) HandleWireMessage(wm WireMessage) {
	switch wm.Kind {
	case KindDirect:
		if err := r.sender.SendTo(wm.RoomID, wm.ToPeer, wm.Envelope); err != nil && !shouldFallbackToBus(err) {
			r.log.Warn("redis direct delivery failed",
				zap.Error(err),
				zap.String("roomID", wm.RoomID),
				zap.String("toPeer", wm.ToPeer))
		}
	case KindBroadcast:
		r.sender.Broadcast(wm.RoomID, wm.ExcludePeer, wm.Envelope)
	}
}

func shouldFallbackToBus(err error) bool {
	return errors.Is(err, room.ErrRoomNotFound) || errors.Is(err, room.ErrPeerNotFound)
}
