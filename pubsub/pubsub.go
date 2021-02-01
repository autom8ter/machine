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
	// If a qgroup is provided, the subscriber will be added to a queue group in which only one subscriber in the group will receive a message per message received
	Subscribe(ctx context.Context, channel, qgroup string, handler Handler) error
	// SubscribeFilter subscribes to messages that pass a given filter
	// If a qgroup is provided, the subscriber will be added to a queue group in which only one subscriber in the group will receive a message per message received
	SubscribeFilter(ctx context.Context, channel, qgroup string, filter Filter, handler Handler) error
	// Close closes all subscriptions
	Close()
}

type pubSub struct {
	subscriptions map[string]map[int]chan interface{}
	qgroups       map[string]map[string]map[int]chan interface{}
	subMu         sync.RWMutex
	closeOnce     sync.Once
}

func NewPubSub() PubSub {
	return &pubSub{
		qgroups:       map[string]map[string]map[int]chan interface{}{},
		subscriptions: map[string]map[int]chan interface{}{},
		subMu:         sync.RWMutex{},
		closeOnce:     sync.Once{},
	}
}

func (p *pubSub) Subscribe(ctx context.Context, channel, qgroup string, handler Handler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel, qgroup)
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

func (p *pubSub) SubscribeFilter(ctx context.Context, channel, qgroup string, decider Filter, handler Handler) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel, qgroup)
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
	channelGroups := p.qgroups[channel]
	if channelGroups != nil {
		for _, input := range channelGroups {
			for _, ch := range input {
				ch <- obj
			}
		}
	}
	if channelSubscribers != nil {
		for _, ch := range channelSubscribers {
			ch <- obj
		}
	}
	return nil
}

func (p *pubSub) setupSubscription(channel, qgroup string) (chan interface{}, func()) {
	subId := rand.Int()
	ch := make(chan interface{}, 10)
	p.subMu.Lock()
	defer p.subMu.Unlock()
	if qgroup == "" {
		if p.subscriptions[channel] == nil {
			p.subscriptions[channel] = map[int]chan interface{}{}
		}
		p.subscriptions[channel][subId] = ch
		return ch, func() {
			p.subMu.Lock()
			delete(p.subscriptions[channel], subId)
			p.subMu.Unlock()
			close(ch)
		}
	} else {
		if p.qgroups[channel] == nil {
			p.qgroups[channel] = map[string]map[int]chan interface{}{}
		}
		if p.qgroups[channel][qgroup] == nil {
			p.qgroups[channel][qgroup] = map[int]chan interface{}{}
		}
		p.qgroups[channel][qgroup][subId] = ch
		return ch, func() {
			p.subMu.Lock()
			delete(p.qgroups[channel][qgroup], subId)
			p.subMu.Unlock()
			close(ch)
		}
	}

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
