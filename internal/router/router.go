package router

import (
	"time"

	"github.com/LessUp/aurora-signal/internal/observability"
	"github.com/LessUp/aurora-signal/internal/signaling"
	redispubsub "github.com/LessUp/aurora-signal/internal/store/redis"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RoomSender defines the interface for sending messages within a room.
type RoomSender interface {
	SendTo(roomID, toPeerID string, env signaling.Envelope) error
	Broadcast(roomID, excludePeerID string, env signaling.Envelope)
}

// Bus defines the interface for cross-node message distribution.
type Bus interface {
	PublishDirect(roomID, toPeer string, env signaling.Envelope) error
	PublishBroadcast(roomID, excludePeer string, env signaling.Envelope) error
}

// Router handles message routing between local and cross-node destinations.
type Router struct {
	sender  RoomSender
	bus     Bus
	log     *zap.Logger
	metrics observability.Metrics
}

// New creates a new message router.
func New(sender RoomSender, bus Bus, log *zap.Logger, metrics observability.Metrics) *Router {
	if metrics == nil {
		metrics = observability.NewNoopMetrics()
	}
	return &Router{
		sender:  sender,
		bus:     bus,
		log:     log,
		metrics: metrics,
	}
}

// Route delivers a message to its destination(s).
// If msg.To is set, it sends directly to that peer (locally first, then via bus if not found).
// Otherwise, it broadcasts to the room (locally and via bus).
func (r *Router) Route(roomID, peerID string, msg signaling.Envelope) {
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

func (r *Router) routeDirect(roomID string, msg signaling.Envelope) {
	if err := r.sender.SendTo(roomID, msg.To, msg); err != nil {
		if r.bus != nil {
			if pubErr := r.bus.PublishDirect(roomID, msg.To, msg); pubErr != nil {
				r.log.Warn("redis direct publish failed",
					zap.Error(pubErr),
					zap.String("roomID", roomID),
					zap.String("toPeer", msg.To))
			}
		}
	}
}

func (r *Router) routeBroadcast(roomID, peerID string, msg signaling.Envelope) {
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
func (r *Router) HandleWireMessage(wm redispubsub.WireMessage) {
	switch wm.Kind {
	case redispubsub.KindDirect:
		_ = r.sender.SendTo(wm.RoomID, wm.ToPeer, wm.Envelope)
	case redispubsub.KindBroadcast:
		r.sender.Broadcast(wm.RoomID, wm.ExcludePeer, wm.Envelope)
	}
}
