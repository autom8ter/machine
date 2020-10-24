package graph

import "github.com/autom8ter/machine/primitive"

// edgeMap is a map of edges. edgeMap are not concurrency safe.
type edgeMap map[string]map[string]*primitive.Edge

func (e edgeMap) Types() []string {
	var typs []string
	for t, _ := range e {
		typs = append(typs, t)
	}
	return typs
}

// RangeType executes the function over a list of edges with the given type. If the function returns false, the iteration stops.
func (e edgeMap) RangeType(typ primitive.Type, fn func(e *primitive.Edge) bool) {
	if e[typ.Type()] == nil {
		return
	}
	for _, e := range e[typ.Type()] {
		if !fn(e) {
			break
		}
	}
}

// Range executes the function over every edge. If the function returns false, the iteration stops.
func (e edgeMap) Range(fn func(e *primitive.Edge) bool) {
	for _, m := range e {
		for _, e := range m {
			if !fn(e) {
				break
			}
		}
	}
}

// Filter executes the function over every edge. If the function returns true, the edges will be added to the returned array of edges.
func (e edgeMap) Filter(fn func(e *primitive.Edge) bool) []*primitive.Edge {
	var edges []*primitive.Edge
	for _, m := range e {
		for _, e := range m {
			if fn(e) {
				edges = append(edges, e)
			}
		}
	}
	return edges
}

// FilterType executes the function over every edge of the given type. If the function returns true, the edges will be added to the returned array of edges.
func (e edgeMap) FilterType(typ primitive.Type, fn func(e *primitive.Edge) bool) []*primitive.Edge {
	var edges []*primitive.Edge
	if e[typ.Type()] == nil {
		return edges
	}
	for _, e := range e[typ.Type()] {
		if fn(e) {
			edges = append(edges, e)
		}
	}
	return edges
}

// DelEdge deletes the edge
func (e edgeMap) DelEdge(id primitive.TypedID) {
	if _, ok := e[id.Type()]; !ok {
		return
	}
	delete(e[id.Type()], id.ID())
}

// AddEdge adds the edge to the map
func (e edgeMap) AddEdge(edge *primitive.Edge) {
	if _, ok := e[edge.Type()]; !ok {
		e[edge.Type()] = map[string]*primitive.Edge{
			edge.ID(): edge,
		}
	} else {
		e[edge.Type()][edge.ID()] = edge
	}
}

// HasEdge returns true if the edge exists
func (e edgeMap) HasEdge(id primitive.TypedID) bool {
	_, ok := e.GetEdge(id)
	return ok
}

// GetEdge gets an edge by id
func (e edgeMap) GetEdge(id primitive.TypedID) (*primitive.Edge, bool) {
	if _, ok := e[id.Type()]; !ok {
		return nil, false
	}
	if e, ok := e[id.Type()][id.ID()]; ok {
		return e, true
	}
	return nil, false
}

// Len returns the number of edges of the given type
func (e edgeMap) Len(typ primitive.Type) int {
	if rels, ok := e[typ.Type()]; ok {
		return len(rels)
	}
	return 0
}
