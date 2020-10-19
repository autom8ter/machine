package graph

// Map is a functional map for storing arbitrary data. It is not concurrency safe
type Map map[interface{}]interface{}

// Exists returns true if the key exists in the map
func (m Map) Exists(key interface{}) bool {
	if m == nil {
		return false
	}
	if _, ok := m[key]; ok {
		return true
	}
	return false
}

// Set set an entry in the map
func (m Map) Set(k, v interface{}) {
	m[k] = v
}

// Get gets an entry from the map by key
func (m Map) Get(k interface{}) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	if !m.Exists(k) {
		return nil, false
	}
	return m[k], true
}

// Del deletes the entry from the map by key
func (m Map) Del(k interface{}) {
	if m == nil {
		return
	}
	delete(m, k)
}

// Range iterates over the map with the function. If the function returns false, the iteration exits.
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

// Filter returns a map of the values that return true from the filter function
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

// Intersection returns the values that exist in both maps ref: https://en.wikipedia.org/wiki/Intersection_(set_theory)#:~:text=In%20mathematics%2C%20the%20intersection%20of,that%20also%20belong%20to%20A).
func (m Map) Intersection(other Map) Map {
	toReturn := Map{}
	m.Range(func(k, v interface{}) bool {
		if other.Exists(k) {
			toReturn.Set(k, v)
		}
		return true
	})
	return toReturn
}

// Union returns the all values in both maps ref: https://en.wikipedia.org/wiki/Union_(set_theory)
func (m Map) Union(other Map) Map {
	toReturn := Map{}
	m.Range(func(k, v interface{}) bool {
		toReturn.Set(k, v)
		return true
	})
	other.Range(func(k, v interface{}) bool {
		toReturn.Set(k, v)
		return true
	})
	return toReturn
}

// Copy creates a replica of the Map
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
