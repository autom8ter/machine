package machine

import (
	"errors"
	"sync"
)

var ErrNoExist = errors.New("machine: does not exit")

// Cache is a concurrency safe cache implementation used by Machine.
// A default sync.Map implementation is used if one isn't provided via WithCache
type Cache interface {
	// Get get a value by key and an error if one exists
	Get(key string) (interface{}, error)
	// Range executes the given function on the cache. If the function returns false, the iteration stops.
	Range(fn func(k string, val interface{}) bool) error
	// Set sets the key and value in the cache
	Set(key string, val interface{}) error
	// Del deletes the value by key from the map
	Del(key string) error
}

type cache struct {
	data *sync.Map
}

func (c *cache) Get(key string) (interface{}, error) {
	val, ok := c.data.Load(key)
	if !ok {
		return val, ErrNoExist
	}
	return val, nil
}

func (c *cache) Range(fn func(k string, val interface{}) bool) error {
	c.data.Range(func(key, value interface{}) bool {
		return fn(key.(string), value)
	})
	return nil
}

func (c *cache) Set(key string, val interface{}) error {
	c.data.Store(key, val)
	return nil
}

func (c *cache) Del(key string) error {
	c.data.Delete(key)
	return nil
}