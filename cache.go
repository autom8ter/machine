package machine

import (
	"context"
	"sync"
	"time"
)

// Cache is a concurrency safe cache that stores arbitrary, namespaced data with optional TTL.
type Cache interface {
	// Get gets the value for the given key in the given namespace.
	Get(namespace string, key interface{}) (interface{}, bool)
	// Set sets a value for the given key in the given namespace with an expiration duration.
	// If the duration is 0 or less, it will be stored forever.
	Set(namespace string, key interface{}, value interface{}, duration time.Duration)
	// Range calls f sequentially for each key and value present within the given namespace.
	// If f returns false, range stops the iteration.
	Range(namespace string, f func(key, value interface{}) bool)
	// Delete deletes the key and its value from the given namespace.
	Delete(namespace string, key interface{})
	// Len returns total kv pairs within namespace
	Len(namespace string) int
	// Close closes the Cache and frees up resources.
	Close()
}

func newCache(ctx context.Context, ticker *time.Ticker) Cache {
	child, cancel := context.WithCancel(ctx)
	n := &namespacedCache{
		cacheMap:  map[string]*cache{},
		mu:        sync.RWMutex{},
		closeOnce: sync.Once{},
		cancel:    cancel,
	}
	go func() {
		defer cancel()
		defer ticker.Stop()
		for {
			select {
			case <-child.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				n.Sync()
			}
		}
	}()
	return n
}

type namespacedCache struct {
	cacheMap  map[string]*cache
	mu        sync.RWMutex
	closeOnce sync.Once
	cancel    func()
}

func (n *namespacedCache) Len(namespace string) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		return c.Len()
	}
	return 0
}

func (n *namespacedCache) Get(namespace string, key interface{}) (interface{}, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		return c.Get(key)
	}
	return nil, false
}

func (n *namespacedCache) Set(namespace string, key interface{}, value interface{}, duration time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.cacheMap[namespace]; !ok {
		n.cacheMap[namespace] = &cache{
			data: sync.Map{},
			once: sync.Once{},
		}
	}
	if c, ok := n.cacheMap[namespace]; ok {
		c.Set(key, value, duration)
	}
}

func (n *namespacedCache) Range(namespace string, f func(key interface{}, value interface{}) bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		c.Range(f)
	}
}

func (n *namespacedCache) Delete(namespace string, key interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if c, ok := n.cacheMap[namespace]; ok {
		c.Delete(key)
	}
}

func (n *namespacedCache) Sync() {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, c := range n.cacheMap {
		c.Sync()
	}
}

func (n *namespacedCache) Close() {
	n.closeOnce.Do(func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		if n.cancel != nil {
			n.cancel()
		}
		for _, c := range n.cacheMap {
			c.Close()
		}
		n.cacheMap = map[string]*cache{}
	})
}

type cache struct {
	data sync.Map
	once sync.Once
}

type item struct {
	data    interface{}
	expires int64
}

func (c *cache) Sync() {
	c.data.Range(func(key, value interface{}) bool {
		now := time.Now().UnixNano()
		item := value.(item)
		if item.expires > 0 && now > item.expires {
			c.data.Delete(key)
		}
		return true
	})
}

func (c *cache) Get(key interface{}) (interface{}, bool) {
	obj, exists := c.data.Load(key)

	if !exists {
		return nil, false
	}

	item := obj.(item)

	if item.expires > 0 && time.Now().UnixNano() > item.expires {
		return nil, false
	}

	return item.data, true
}

func (c *cache) Set(key interface{}, value interface{}, duration time.Duration) {
	var expires int64

	if duration > 0 {
		expires = time.Now().Add(duration).UnixNano()
	}

	c.data.Store(key, item{
		data:    value,
		expires: expires,
	})
}

func (c *cache) Range(f func(key, value interface{}) bool) {
	now := time.Now().UnixNano()
	c.data.Range(func(key, value interface{}) bool {
		item := value.(item)

		if item.expires > 0 && now > item.expires {
			return true
		}

		return f(key, item.data)
	})
}

func (c *cache) Delete(key interface{}) {
	c.data.Delete(key)
}

func (c *cache) Len() int {
	i := 0
	c.data.Range(func(key, value interface{}) bool {
		i++
		return true
	})
	return i
}

func (c *cache) Close() {
	c.once.Do(func() {
		c.Sync()
		c.data = sync.Map{}
	})
}
