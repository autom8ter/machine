package machine

import (
	"errors"
	"sync"
)

var ErrNoExist = errors.New("machine: does not exit")

type Cache interface {
	Get(key string) (interface{}, error)
	Range(fn func(k string, val interface{}) bool) error
	Set(key string, val interface{}) error
	Del(key string) error
}

type cache struct {
	data *sync.Map
}

func (c cache) Get(key string) (interface{}, error) {
	val, ok := c.data.Load(key)
	if !ok {
		return val, ErrNoExist
	}
	return val, nil
}

func (c cache) Range(fn func(k string, val interface{}) bool) error {
	c.data.Range(func(key, value interface{}) bool {
		return fn(key.(string), value)
	})
	return nil
}

func (c cache) Set(key string, val interface{}) error {
	c.data.Store(key, val)
	return nil
}

func (c cache) Del(key string) error {
	c.data.Delete(key)
	return nil
}
