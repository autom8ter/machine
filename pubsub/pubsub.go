package pubsub

import (
	"context"
	"math/rand"
	"sync"
)

// PubSub is used to asynchronously pass messages between routines.
type PubSub interface {
	// Publish publishes the object to the channel by name
	Publish(channel string, obj interface{}) error
	// PublishN publishes the object to the channel by name to the first N subscribers of the channel
	PublishN(channel string, obj interface{}, n int) error
	// Subscribe subscribes to the given channel until the context is cancelled
	Subscribe(ctx context.Context, channel string, handler func(obj interface{})) error
	// Subscribe subscribes to the given channel until it receives N messages or the context is cancelled
	SubscribeN(ctx context.Context, channel string, n int, handler func(msg interface{})) error
	// SubscribeUntil subscribes to the given channel until the decider returns false for the first time. The subscription breaks when the routine's context is cancelled or the decider returns false.
	SubscribeUntil(ctx context.Context, channel string, decider func() bool, handler func(msg interface{})) error
	// SubscribeWhile subscribes to the given channel while the decider returns true. The subscription breaks when the routine's context is cancelled.
	SubscribeWhile(ctx context.Context, channel string, decider func() bool, handler func(msg interface{})) error
	// SubscribeFilter subscribes to the given channel with the given filter
	SubscribeFilter(ctx context.Context, channel string, filter func(msg interface{}) bool, handler func(msg interface{})) error
	Close()
}

type pubSub struct {
	subscriptions map[string]map[int]chan interface{}
	subMu         sync.RWMutex
	closeOnce     sync.Once
}

func NewPubSub() PubSub {
	return &pubSub{
		subscriptions: map[string]map[int]chan interface{}{},
		subMu:         sync.RWMutex{},
		closeOnce:     sync.Once{},
	}
}

func (p *pubSub) Subscribe(ctx context.Context, channel string, handler func(msg interface{})) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			handler(msg)
		}
	}
}

func (p *pubSub) SubscribeN(ctx context.Context, channel string, n int, handler func(msg interface{})) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	count := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			count++
			handler(msg)
			if count >= n {
				return nil
			}
		}
	}
}

func (p *pubSub) SubscribeUntil(ctx context.Context, channel string, decider func() bool, handler func(msg interface{})) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			if !decider() {
				return nil
			}
			handler(msg)
		}
	}
}

func (p *pubSub) SubscribeWhile(ctx context.Context, channel string, decider func() bool, handler func(msg interface{})) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			if decider() {
				handler(msg)
			}
		}
	}
}

func (p *pubSub) SubscribeFilter(ctx context.Context, channel string, decider func(msg interface{}) bool, handler func(msg interface{})) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			if decider(msg) {
				handler(msg)
			}
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

func (p *pubSub) PublishN(channel string, obj interface{}, n int) error {
	p.subMu.Lock()
	defer p.subMu.Unlock()
	if p.subscriptions[channel] == nil {
		p.subscriptions[channel] = map[int]chan interface{}{}
	}
	channelSubscribers := p.subscriptions[channel]
	count := 0
	for _, input := range channelSubscribers {
		if count >= n {
			continue
		}
		input <- obj
		count++
	}
	return nil
}

func (p *pubSub) setupSubscription(channel string) (chan interface{}, func()) {
	p.subMu.Lock()
	if p.subscriptions[channel] == nil {
		p.subscriptions[channel] = map[int]chan interface{}{}
	}
	ch := make(chan interface{}, 10)
	subId := rand.Int()
	p.subscriptions[channel][subId] = ch
	p.subMu.Unlock()
	return ch, func() {
		p.subMu.Lock()
		delete(p.subscriptions[channel], subId)
		p.subMu.Unlock()
		close(ch)
	}
}

func (p *pubSub) Close() {
	p.closeOnce.Do(func() {
		p.subMu.Lock()
		defer p.subMu.Unlock()
		for k, v := range p.subscriptions {
			delete(p.subscriptions, k)
			for _, v2 := range v {
				close(v2)
			}
		}
	})
}
