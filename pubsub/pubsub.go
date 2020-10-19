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
	// Publish publishes the object to the channel by name to the first N subscribers of the channel
	PublishN(channel string, obj interface{}, n int) error
	// Subscribe subscribes to the given channel until the context is cancelled
	Subscribe(ctx context.Context, channel string, handler func(obj interface{})) error
	// Subscribe subscribes to the given channel until it receives N messages or the context is cancelled
	SubscribeN(ctx context.Context, channel string, n int, handler func(msg interface{})) error
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
		close(ch)
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

func (p *pubSub) SubscribeN(ctx context.Context, channel string, n int, handler func(msg interface{})) error {
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
		close(ch)
	}()
	count := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			handler(msg)
			count++
			if count >= n {
				cancel()
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
