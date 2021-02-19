package pubsub

import (
	"context"
	"math/rand"
	"sync"
)

// Handler is the function executed against the inbound message in a subscription
type Handler func(msg interface{}) bool

// Filter is a function to filter out messages before they reach a subscriptions primary Handler
type Filter func(msg interface{}) bool

// PubSub is used to asynchronously pass messages between routines.
type PubSub interface {
	// Publish publishes the object to the channel by name
	Publish(channel string, obj interface{}) error
	// Subscribe subscribes to the given channel until the context is cancelled.
	Subscribe(ctx context.Context, channel string, handler Handler) error
	// SubscribeFilter subscribes to messages that pass a given filter
	SubscribeFilter(ctx context.Context, channel string, filter Filter, handler Handler) error
	// Close closes all subscriptions
	Close()
}

type pubSub struct {
	subscriptions    map[string]map[int]chan interface{}
	gobSubscriptions map[string]map[int]chan interface{}
	subMu            sync.RWMutex
}

func NewPubSub() PubSub {
	return &pubSub{
		subscriptions:    map[string]map[int]chan interface{}{},
		gobSubscriptions: nil,
		subMu:            sync.RWMutex{},
	}
}

func (p *pubSub) Subscribe(ctx context.Context, channel string, handler Handler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			if !handler(msg) {
				return nil
			}
		}
	}
}

func (p *pubSub) SubscribeFilter(ctx context.Context, channel string, decider Filter, handler Handler) error {
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
				if !handler(msg) {
					return nil
				}
			}
		}
	}
}

func (p *pubSub) Publish(channel string, obj interface{}) error {
	p.subMu.RLock()
	defer p.subMu.RUnlock()
	if p.subscriptions[channel] == nil {
		return nil
	}
	channelSubscribers := p.subscriptions[channel]
	if channelSubscribers != nil {
		for _, ch := range channelSubscribers {
			ch <- obj
		}
	}
	return nil
}

func (p *pubSub) setupSubscription(channel string) (chan interface{}, func()) {
	subId := rand.Int()
	ch := make(chan interface{}, 1)
	p.subMu.Lock()
	if p.subscriptions[channel] == nil {
		p.subscriptions[channel] = map[int]chan interface{}{}
	}
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
	p.subMu.Lock()
	for k, _ := range p.subscriptions {
		delete(p.subscriptions, k)
	}
	p.subMu.Unlock()
}
