package graph

import (
	"sync"
)

func newCache() *namespacedCache {
	return &namespacedCache{
		cacheMap:  map[string]*cache{},
		mu:        sync.RWMutex{},
		closeOnce: sync.Once{},
	}
}

type namespacedCache struct {
	cacheMap  map[string]*cache
	mu        sync.RWMutex
	closeOnce sync.Once
}

func (n *namespacedCache) Len(namespace string) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		return c.Len()
	}
	return 0
}

func (n *namespacedCache) Namespaces() []string {
	var namespaces []string
	n.mu.RLock()
	defer n.mu.RUnlock()
	for k, _ := range n.cacheMap {
		namespaces = append(namespaces, k)
	}
	return namespaces
}

func (n *namespacedCache) Get(namespace string, key interface{}) (interface{}, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		return c.Get(key)
	}
	return nil, false
}

func (n *namespacedCache) Set(namespace string, key interface{}, value interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.cacheMap[namespace]; !ok {
		n.cacheMap[namespace] = &cache{
			data: sync.Map{},
			once: sync.Once{},
		}

	}
	n.cacheMap[namespace].Set(key, value)
}

func (n *namespacedCache) Range(namespace string, f func(key interface{}, value interface{}) bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		c.Range(f)
	}
}

func (n *namespacedCache) Delete(namespace string, key interface{}) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if c, ok := n.cacheMap[namespace]; ok {
		c.Delete(key)
	}
}

func (n *namespacedCache) Exists(namespace string, key interface{}) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace].Exists(key)
}

func (n *namespacedCache) Copy(namespace string) Map {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace].Copy()
}

func (n *namespacedCache) Filter(namespace string, filter func(k, v interface{}) bool) Map {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace].Filter(filter)
}

func (n *namespacedCache) Intersection(namespace1, namespace2 string) Map {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace1].Intersection(n.cacheMap[namespace2])
}

func (n *namespacedCache) Union(namespace1, namespace2 string) Map {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace1].Union(n.cacheMap[namespace2])
}

func (n *namespacedCache) Map(namespace string) Map {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.cacheMap[namespace].Map()
}

func (n *namespacedCache) SetAll(namespace string, m Map) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.cacheMap[namespace]; !ok {
		n.cacheMap[namespace] = &cache{
			data: sync.Map{},
			once: sync.Once{},
		}

	}
	n.cacheMap[namespace].SetAll(m)
}

func (n *namespacedCache) Clear(namespace string) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if cache, ok := n.cacheMap[namespace]; ok {
		cache.Clear()
	}
}

func (n *namespacedCache) Close() {
	n.closeOnce.Do(func() {
		n.mu.Lock()
		defer n.mu.Unlock()
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

func (c *cache) Get(key interface{}) (interface{}, bool) {
	obj, exists := c.data.Load(key)

	if !exists {
		return nil, false
	}
	return obj, true
}

func (c *cache) Set(key interface{}, value interface{}) {
	c.data.Store(key, value)
}

func (c *cache) Range(f func(key, value interface{}) bool) {
	c.data.Range(func(key, value interface{}) bool {
		return f(key, value)
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

func (c *cache) Exists(key interface{}) bool {
	_, ok := c.Get(key)
	return ok
}

func (c *cache) Close() {
	c.once.Do(func() {
		c.data = sync.Map{}
	})
}

func (c *cache) Map() Map {
	data := make(map[interface{}]interface{})
	c.Range(func(key, value interface{}) bool {
		data[key] = value
		return true
	})
	return data
}

func (c *cache) SetAll(m Map) {
	m.Range(func(k, v interface{}) bool {
		c.Set(k, v)
		return true
	})
}

func (c *cache) Intersection(other *cache) Map {
	data := Map{}
	if c == nil {
		return data
	}
	if other != nil {
		c.Range(func(k, v interface{}) bool {
			if other.Exists(v) {
				data.Set(k, v)
			}
			return true
		})
	}
	return data
}

func (c *cache) Union(other *cache) Map {
	data := Map{}
	if c != nil {
		c.Range(func(k, v interface{}) bool {
			data.Set(k, v)
			return true
		})
	}
	if other != nil {
		other.Range(func(k, v interface{}) bool {
			data.Set(k, v)
			return true
		})
	}
	return data
}

func (c *cache) Copy() Map {
	data := Map{}
	if c == nil {
		return data
	}
	c.Range(func(k, v interface{}) bool {
		data.Set(k, v)
		return true
	})
	return data
}

func (c *cache) Filter(filter func(k, v interface{}) bool) Map {
	data := Map{}
	if c == nil {
		return data
	}
	c.Range(func(key, value interface{}) bool {
		if filter(key, value) {
			data.Set(key, value)
		}
		return true
	})
	return data
}

func (c *cache) Clear() {
	c.Range(func(key, value interface{}) bool {
		c.Delete(key)
		return true
	})
}
