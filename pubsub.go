package machine

import (
	"context"
	"math/rand"
	"sync"
)

// PubSub is used to asynchronously pass messages between routines.
type PubSub interface {
	// Publish publishes the object to the channel by name
	Publish(channel string, obj interface{}) error
	// Subscribe subscribes to the given channel
	Subscribe(ctx context.Context, channel string, handler func(obj interface{})) error
	Close()
}

type pubSub struct {
	subscriptions map[string]map[int]chan interface{}
	subMu         sync.RWMutex
	closeOnce     sync.Once
}

func (p *pubSub) Subscribe(ctx context.Context, channel string, handler func(msg interface{})) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p.subMu.Lock()
	if p.subscriptions[channel] == nil {
		p.subscriptions[channel] = map[int]chan interface{}{}
	}
	ch := make(chan interface{}, 10)
	subId := rand.Int()
	p.subscriptions[channel][subId] = ch
	p.subMu.Unlock()
	defer func() {
		p.subMu.Lock()
		delete(p.subscriptions[channel], subId)
		p.subMu.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			handler(msg)
		}
	}
}

func (p *pubSub) Publish(channel string, obj interface{}) error {
	p.subMu.Lock()
	defer p.subMu.Unlock()
	if p.subscriptions[channel] == nil {
		p.subscriptions[channel] = map[int]chan interface{}{}
	}
	channelSubscribers := p.subscriptions[channel]
	for _, input := range channelSubscribers {
		input <- obj
	}
	return nil
}

func (p *pubSub) Close() {
	p.closeOnce.Do(func() {
		p.subMu.Lock()
		defer p.subMu.Unlock()
		for k, _ := range p.subscriptions {
			delete(p.subscriptions, k)
		}
	})
}
