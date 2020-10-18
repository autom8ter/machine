package machine

import (
	"encoding/json"
	"fmt"
)

// Map is a functional map for storing arbitrary data. It is not concurrency safe
type Map map[interface{}]interface{}

func (m Map) Exists(key interface{}) bool {
	if m == nil {
		return false
	}
	if _, ok := m[key]; ok {
		return true
	}
	return false
}

func (m Map) Set(k, v interface{}) {
	m[k] = v
}

func (m Map) Get(k interface{}) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	if !m.Exists(k) {
		return nil, false
	}
	return m[k], true
}

func (m Map) Del(k interface{}) {
	if m == nil {
		return
	}
	delete(m, k)
}

func (m Map) Range(iterator func(k, v interface{}) bool) {
	if m == nil {
		return
	}
	for k, v := range m {
		if !iterator(k, v) {
			break
		}
	}
}

func (m Map) Filter(filter func(k, v interface{}) bool) Map {
	data := Map{}
	if m == nil {
		return data
	}
	m.Range(func(k, v interface{}) bool {
		if filter(k, v) {
			data.Set(k, v)
		}
		return true
	})
	return data
}

func (m Map) Intersection(k interface{}) (interface{}, bool) {
	if !m.Exists(k) {
		return nil, false
	}
	return m[k], true
}

func (m Map) Copy() Map {
	copied := Map{}
	if m == nil {
		return copied
	}
	m.Range(func(k, v interface{}) bool {
		copied.Set(k, v)
		return true
	})
	return copied
}

func (m Map) stringMap() map[string]interface{} {
	strMap := map[string]interface{}{}
	m.Range(func(k, v interface{}) bool {
		if s, ok := k.(string); ok {
			strMap[s] = v
			return true
		}
		if s, ok := k.(fmt.Stringer); ok {
			strMap[s.String()] = v
			return true
		}
		strMap[fmt.Sprintf("%v", k)] = v
		return true
	})
	return strMap
}

func (m Map) Marshal() ([]byte, error) {
	return json.Marshal(m.stringMap())
}
