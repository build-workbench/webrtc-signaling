package redispubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/LessUp/aurora-signal/internal/signaling"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MessageKind string

const (
	KindBroadcast MessageKind = "broadcast"
	KindDirect    MessageKind = "direct"
)

type WireMessage struct {
	Kind        MessageKind        `json:"kind"`
	RoomID      string             `json:"roomId"`
	ToPeer      string             `json:"toPeer,omitempty"`
	ExcludePeer string             `json:"excludePeer,omitempty"`
	Envelope    signaling.Envelope `json:"envelope"`
	Origin      string             `json:"origin"`
}

type Bus struct {
	client *redis.Client
	log    *zap.Logger
	nodeID string
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	subs   map[string]*redis.PubSub
}

func New(addr, password string, db int, nodeID string, log *zap.Logger) (*Bus, error) {
	cli := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := cli.Ping(pingCtx).Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{client: cli, nodeID: nodeID, log: log, ctx: ctx, cancel: cancel, subs: map[string]*redis.PubSub{}}, nil
}

func (b *Bus) Ping() error {
	ctx, cancel := context.WithTimeout(b.ctx, 3*time.Second)
	defer cancel()
	return b.client.Ping(ctx).Err()
}

func (b *Bus) channel(roomID string) string { return fmt.Sprintf("chan:room:%s", roomID) }

func (b *Bus) PublishBroadcast(roomID, excludePeer string, env signaling.Envelope) error {
	msg := WireMessage{Kind: KindBroadcast, RoomID: roomID, ExcludePeer: excludePeer, Envelope: env, Origin: b.nodeID}
	return b.publish(msg)
}

func (b *Bus) PublishDirect(roomID, toPeer string, env signaling.Envelope) error {
	msg := WireMessage{Kind: KindDirect, RoomID: roomID, ToPeer: toPeer, Envelope: env, Origin: b.nodeID}
	return b.publish(msg)
}

func (b *Bus) publish(m WireMessage) error {
	ch := b.channel(m.RoomID)
	data, _ := json.Marshal(m)
	return b.client.Publish(b.ctx, ch, data).Err()
}

func (b *Bus) SubscribeRoom(roomID string, handler func(WireMessage)) error {
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

func (b *Bus) UnsubscribeRoom(roomID string) error {
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

func (b *Bus) Close() error {
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
