package machine

import (
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
	// Sync performs any cleanup/sync operations like garbage collection & persistance
	Sync()
	// Close closes the Cache and frees up resources.
	Close()
}

func newCache() Cache {
	return &namespacedCache{
		mu:       sync.RWMutex{},
		cacheMap: map[string]*cache{},
	}
}

type namespacedCache struct {
	mu       sync.RWMutex
	cacheMap map[string]*cache
}

func (n *namespacedCache) initNamespace(namespace string) {
	if n.cacheMap == nil {
		n.mu.Lock()
		n.cacheMap = map[string]*cache{}
		n.mu.Unlock()
	}
	if _, ok := n.cacheMap[namespace]; !ok {
		n.mu.Lock()
		n.cacheMap[namespace] = &cache{
			items: sync.Map{},
			once:  sync.Once{},
		}
		n.mu.Unlock()
	}
}

func (n *namespacedCache) Get(namespace string, key interface{}) (interface{}, bool) {
	n.initNamespace(namespace)
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace].Get(key)
}

func (n *namespacedCache) Set(namespace string, key interface{}, value interface{}, duration time.Duration) {
	n.initNamespace(namespace)
	n.mu.RLock()
	defer n.mu.RUnlock()
	n.cacheMap[namespace].Set(key, value, duration)
}

func (n *namespacedCache) Range(namespace string, f func(key interface{}, value interface{}) bool) {
	n.initNamespace(namespace)
	n.mu.RLock()
	defer n.mu.RUnlock()
	n.cacheMap[namespace].Range(f)
}

func (n *namespacedCache) Delete(namespace string, key interface{}) {
	n.initNamespace(namespace)
	n.mu.RLock()
	defer n.mu.RUnlock()
	n.cacheMap[namespace].Delete(key)
}

func (n *namespacedCache) Sync() {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, c := range n.cacheMap {
		c.Sync()
	}
}

func (n *namespacedCache) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, c := range n.cacheMap {
		c.Close()
	}
	n.cacheMap = map[string]*cache{}
}

type cache struct {
	items sync.Map
	once  sync.Once
}

type item struct {
	data    interface{}
	expires int64
}

func (c *cache) Sync() {
	now := time.Now().UnixNano()
	c.items.Range(func(key, value interface{}) bool {
		item := value.(item)

		if item.expires > 0 && now > item.expires {
			c.items.Delete(key)
		}

		return true
	})
}

func (c *cache) Get(key interface{}) (interface{}, bool) {
	obj, exists := c.items.Load(key)

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

	c.items.Store(key, item{
		data:    value,
		expires: expires,
	})
}

func (c *cache) Range(f func(key, value interface{}) bool) {
	now := time.Now().UnixNano()
	c.items.Range(func(key, value interface{}) bool {
		item := value.(item)

		if item.expires > 0 && now > item.expires {
			return true
		}

		return f(key, item.data)
	})
}

func (c *cache) Delete(key interface{}) {
	c.items.Delete(key)
}

func (c *cache) Close() {
	c.once.Do(func() {
		c.items = sync.Map{}
	})
}
