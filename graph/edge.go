package graph

// Edge is a relationship between two nodes
type Edge interface {
	// An edge implements Node because it has an Identifier and attributes
	Node
	// From returns the root node of the edge
	From() Node
	// To returns the target node of the edge
	To() Node
}

type edge struct {
	Node
	from Node
	to   Node
}

func (e *edge) From() Node {
	return e.from
}

func (e *edge) To() Node {
	return e.to
}

// Edges is a map of edges. Edges are not concurrency safe.
type Edges map[string]map[string]Edge

func (e Edges) Types() []string {
	var typs []string
	for t, _ := range e {
		typs = append(typs, t)
	}
	return typs
}

// RangeType executes the function over a list of edges with the given type. If the function returns false, the iteration stops.
func (e Edges) RangeType(typ string, fn func(e Edge) bool) {
	if e[typ] == nil {
		return
	}
	for _, e := range e[typ] {
		if !fn(e) {
			break
		}
	}
}

// Range executes the function over every edge. If the function returns false, the iteration stops.
func (e Edges) Range(fn func(e Edge) bool) {
	for _, m := range e {
		for _, e := range m {
			if !fn(e) {
				break
			}
		}
	}
}

// DelEdge deletes the edge
func (e Edges) DelEdge(id ID) {
	if _, ok := e[id.Type()]; !ok {
		return
	}
	delete(e[id.Type()], id.ID())
}

// AddEdge adds the edge to the map
func (e Edges) AddEdge(edge Edge) {
	if _, ok := e[edge.Type()]; !ok {
		e[edge.Type()] = map[string]Edge{
			edge.ID(): edge,
		}
	} else {
		e[edge.Type()][edge.ID()] = edge
	}
}

// HasEdge returns true if the edge exists
func (e Edges) HasEdge(id ID) bool {
	_, ok := e.GetEdge(id)
	return ok
}

// GetEdge gets an edge by id
func (e Edges) GetEdge(id ID) (Edge, bool) {
	if _, ok := e[id.Type()]; !ok {
		return nil, false
	}
	if e, ok := e[id.Type()][id.ID()]; ok {
		return e, true
	}
	return nil, false
}

// Len returns the number of edges of the given type
func (e Edges) Len(typ string) int {
	if rels, ok := e[typ]; ok {
		return len(rels)
	}
	return 0
}
