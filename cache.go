package machine

import (
	"fmt"
	"sync"
)

type Cache interface {
	Get(id string) (map[string]interface{}, error)
	Set(id string, data map[string]interface{}) error
	Range(fn func(key string, data map[string]interface{}) bool)
	Sync() error
	Del(id string) error
	Close() error
}

type inMem struct {
	data *sync.Map
}

func NewInMemStorage() Cache {
	return &inMem{data: &sync.Map{}}
}

func (i *inMem) Get(id string) (map[string]interface{}, error) {
	val, ok := i.data.Load(id)
	if !ok {
		return nil, fmt.Errorf("%s does not exist", id)
	}
	return val.(map[string]interface{}), nil
}

func (i *inMem) Sync() error {
	i.Range(func(id string, data map[string]interface{}) bool {
		if data["expired"] == true {
			if err := i.Del(id); err != nil {
				i.Del(id)
			}
		}
		if data["stale"] == true {
			i.Del(id)
		}
		return true
	})
	return nil
}

func (i *inMem) Set(id string, data map[string]interface{}) error {
	i.data.Store(id, data)
	return nil
}

func (i *inMem) Range(fn func(id string, data map[string]interface{}) bool) {
	i.data.Range(func(key, value interface{}) bool {
		return fn(key.(string), value.(map[string]interface{}))
	})
}

func (i *inMem) Del(id string) error {
	i.data.Delete(id)
	return nil
}

func (i *inMem) Close() error {
	return nil
}
