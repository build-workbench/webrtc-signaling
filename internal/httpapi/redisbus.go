package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/vibe-knight/webrtc-signaling/internal/room"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MessageKind string

const (
	KindBroadcast MessageKind = "broadcast"
	KindDirect    MessageKind = "direct"
)

// WireMessage is the envelope used for cross-node pub/sub.
type WireMessage struct {
	Kind        MessageKind     `json:"kind"`
	RoomID      string          `json:"roomId"`
	ToPeer      string          `json:"toPeer,omitempty"`
	ExcludePeer string          `json:"excludePeer,omitempty"`
	Envelope    room.Envelope   `json:"envelope"`
	Origin      string          `json:"origin"`
}

// RedisBus implements Bus via Redis Pub/Sub.
type RedisBus struct {
	client *redis.Client
	log    *zap.Logger
	nodeID string
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	subs   map[string]*redis.PubSub
}

func NewRedisBus(addr, nodeID string, log *zap.Logger) (*RedisBus, error) {
	cli := redis.NewClient(&redis.Options{Addr: addr})
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := cli.Ping(pingCtx).Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisBus{
		client: cli, nodeID: nodeID, log: log, ctx: ctx, cancel: cancel, subs: map[string]*redis.PubSub{},
	}, nil
}

func (b *RedisBus) Ping() error {
	ctx, cancel := context.WithTimeout(b.ctx, 3*time.Second)
	defer cancel()
	return b.client.Ping(ctx).Err()
}

func (b *RedisBus) channel(roomID string) string { return fmt.Sprintf("chan:room:%s", roomID) }

func (b *RedisBus) PublishBroadcast(roomID, excludePeer string, env room.Envelope) error {
	return b.publish(WireMessage{Kind: KindBroadcast, RoomID: roomID, ExcludePeer: excludePeer, Envelope: env, Origin: b.nodeID})
}

func (b *RedisBus) PublishDirect(roomID, toPeer string, env room.Envelope) error {
	return b.publish(WireMessage{Kind: KindDirect, RoomID: roomID, ToPeer: toPeer, Envelope: env, Origin: b.nodeID})
}

func (b *RedisBus) publish(m WireMessage) error {
	data, _ := json.Marshal(m)
	return b.client.Publish(b.ctx, b.channel(m.RoomID), data).Err()
}

func (b *RedisBus) SubscribeRoom(roomID string, handler func(WireMessage)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subs[roomID]; ok {
		return nil
	}
	ps := b.client.Subscribe(b.ctx, b.channel(roomID))
	b.subs[roomID] = ps
	go func() {
		for msg := range ps.Channel() {
			var wm WireMessage
			if err := json.Unmarshal([]byte(msg.Payload), &wm); err != nil {
				b.log.Warn("redis unmarshal", zap.Error(err))
				continue
			}
			if wm.Origin == b.nodeID {
				continue // ignore self
			}
			handler(wm)
		}
	}()
	return nil
}

func (b *RedisBus) UnsubscribeRoom(roomID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	ps, ok := b.subs[roomID]
	if !ok {
		return nil
	}
	delete(b.subs, roomID)
	if err := ps.Unsubscribe(b.ctx, b.channel(roomID)); err != nil {
		_ = ps.Close()
		return err
	}
	return ps.Close()
}

func (b *RedisBus) Close() error {
	b.cancel()
	b.mu.Lock()
	defer b.mu.Unlock()
	for roomID, ps := range b.subs {
		_ = ps.Unsubscribe(b.ctx, b.channel(roomID))
		_ = ps.Close()
	}
	b.subs = map[string]*redis.PubSub{}
	return b.client.Close()
}
