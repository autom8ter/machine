package pubsub

import (
	"context"
	"math/rand"
	"strings"
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
	Subscribe(ctx context.Context, channel string, handler Handler, opts ...SubOpt) error
	// Close closes all subscriptions
	Close()
}

type SubOptions struct {
	filter Filter
}

type SubOpt func(options *SubOptions)

func WithFilter(filter Filter) SubOpt {
	return func(options *SubOptions) {
		options.filter = filter
	}
}

type pubSub struct {
	subscriptions map[string]map[int]chan interface{}
	subMu         sync.RWMutex
}

func NewPubSub() PubSub {
	return &pubSub{
		subscriptions: map[string]map[int]chan interface{}{},
		subMu:         sync.RWMutex{},
	}
}

func (p *pubSub) Subscribe(ctx context.Context, channel string, handler Handler, options ...SubOpt) error {
	opts := &SubOptions{}
	for _, o := range options {
		o(opts)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ch, closer := p.setupSubscription(channel)
	defer closer()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			if opts.filter != nil && !opts.filter(msg) {
				continue
			}
			if !handler(msg) {
				return nil
			}
		}
	}
}

func (p *pubSub) Publish(channel string, obj interface{}) error {
	p.subMu.RLock()
	defer p.subMu.RUnlock()
	for k, channelSubscribers := range p.subscriptions {
		if globMatch(k, channel) {
			for _, ch := range channelSubscribers {
				ch <- obj
			}
		}
	}
	return nil
}

func (p *pubSub) setupSubscription(channel string) (chan interface{}, func()) {
	subId := rand.Int()
	ch := make(chan interface{}, 10)
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
	defer p.subMu.Unlock()
	for k, _ := range p.subscriptions {
		delete(p.subscriptions, k)
	}
}

func globMatch(pattern string, subj string) bool {
	const matchAll = "*"
	if pattern == subj {
		return true
	}
	if pattern == matchAll {
		return true
	}

	parts := strings.Split(pattern, matchAll)
	if len(parts) == 1 {
		return subj == pattern
	}
	leadingGlob := strings.HasPrefix(pattern, matchAll)
	trailingGlob := strings.HasSuffix(pattern, matchAll)
	end := len(parts) - 1
	for i := 0; i < end; i++ {
		idx := strings.Index(subj, parts[i])

		switch i {
		case 0:
			if !leadingGlob && idx != 0 {
				return false
			}
		default:
			if idx < 0 {
				return false
			}
		}
		subj = subj[idx+len(parts[i]):]
	}
	return trailingGlob || strings.HasSuffix(subj, parts[end])
}
