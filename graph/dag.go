package graph

import (
	"fmt"
)

// Graph is a concurrency safe directed Graph datastructure
type Graph interface {
	// NewID is a constructor for a new ID. If an id is not specified, a random one will be generated automatically.
	NewID(typ string, id string) ID
	// NewEdge is a constructor for a new graph edge node
	NewEdge(id ID, attributes Map, from, to Node) Edge
	// NewNode is a constructor for a a new graph node
	NewNode(id ID, attributes Map) Node
	// AddNode adds a single node to the graph
	AddNode(n Node)
	// GetNode gets a node from the graph if it exists
	GetNode(id ID) (Node, bool)
	// DelNode deletes the nodes and it's edges
	DelNode(id ID)
	// HasNode returns true if the node exists in the graph
	HasNode(id ID) bool
	// NodeTypes returns an array of Node types that exist in the graph
	NodeTypes() []string
	// RangeNodeTypes ranges over every node of the given type until the given function returns false
	RangeNodeTypes(typ string, fn func(n Node) bool)
	// RangeNodes ranges over every node until the given function returns false
	RangeNodes(fn func(n Node) bool)
	// AddEdge adds an edge to the graph between the two nodes. It returns an error if one of the nodes does not exist.
	AddEdge(e Edge) error
	// AddEdges adds all of the edges to the graph
	AddEdges(e Edges) error
	// GetEdge gets the edge by id if it exists
	GetEdge(id ID) (Edge, bool)
	// HasEdge returns true if the edge exists in the graph
	HasEdge(id ID) bool
	// DelEdge deletes the edge
	DelEdge(id ID)
	// DelEdges deletes all the edges at once
	DelEdges(e Edges) error
	// RangeEdges ranges over every edge until the given function returns false
	RangeEdges(fn func(e Edge) bool)
	// RangeEdgeTypes ranges over every edge of the given type until the given function returns false
	RangeEdgeTypes(typ string, fn func(e Edge) bool)
	// EdgesFrom returns all of the edges the point from the given node identifier
	EdgesFrom(id ID) (Edges, bool)
	// EdgesTo returns all of the edges the point to the given node identifier
	EdgesTo(id ID) (Edges, bool)
	// EdgeTypes returns all of the edge types in the Graph
	EdgeTypes() []string
	// Close removes all entries from the cache
	Close()
}

type graph struct {
	nodes     *namespacedCache
	edges     *namespacedCache
	edgesFrom *namespacedCache
	edgesTo   *namespacedCache
}

func NewGraph() Graph {
	return &graph{
		nodes:     newCache(),
		edges:     newCache(),
		edgesFrom: newCache(),
		edgesTo:   newCache(),
	}
}

func (g *graph) AddNode(n Node) {
	g.nodes.Set(n.Type(), n.ID(), n)
}

func (g *graph) GetNode(id ID) (Node, bool) {
	val, ok := g.nodes.Get(id.Type(), id.ID())
	if ok {
		n, ok := val.(Node)
		if ok {
			return n, true
		}
	}
	return nil, false
}

func (g *graph) RangeNodeTypes(typ string, fn func(n Node) bool) {
	g.nodes.Range(typ, func(key interface{}, val interface{}) bool {
		n, ok := val.(Node)
		if ok {
			if !fn(n) {
				return false
			}
		}
		return true
	})
}

func (g *graph) RangeNodes(fn func(n Node) bool) {
	for _, namespace := range g.nodes.Namespaces() {
		g.nodes.Range(namespace, func(key interface{}, val interface{}) bool {
			n, ok := val.(Node)
			if ok {
				if !fn(n) {
					return false
				}
			}
			return true
		})
	}
}

func (g *graph) RangeEdges(fn func(e Edge) bool) {
	for _, namespace := range g.edges.Namespaces() {
		g.edges.Range(namespace, func(key interface{}, val interface{}) bool {
			e, ok := val.(Edge)
			if ok {
				if !fn(e) {
					return false
				}
			}
			return true
		})
	}
}

func (g *graph) RangeEdgeTypes(typ string, fn func(e Edge) bool) {
	g.edges.Range(typ, func(key interface{}, val interface{}) bool {
		e, ok := val.(Edge)
		if ok {
			if !fn(e) {
				return false
			}
		}
		return true
	})
}

func (g *graph) HasNode(id ID) bool {
	_, ok := g.GetNode(id)
	return ok
}

func (g *graph) DelNode(id ID) {
	if val, ok := g.edgesFrom.Get(id.Type(), id.ID()); ok {
		if val != nil {
			edges := val.(Edges)
			edges.Range(func(e Edge) bool {
				g.DelEdge(e)
				return true
			})
		}
	}
	g.nodes.Delete(id.Type(), id.ID())
}

func (g *graph) AddEdge(e Edge) error {
	if !g.HasNode(e.From()) {
		return fmt.Errorf("node %s does not exist", e.From().String())
	}
	if !g.HasNode(e.To()) {
		return fmt.Errorf("node %s does not exist", e.To().String())
	}
	g.edges.Set(e.Type(), e.ID(), e)
	if val, ok := g.edgesFrom.Get(e.From().Type(), e.From().ID()); ok {
		edges := val.(Edges)
		edges.AddEdge(e)
	} else {
		edges := Edges{}
		edges.AddEdge(e)
		g.edgesFrom.Set(e.From().Type(), e.From().ID(), edges)
	}
	if val, ok := g.edgesTo.Get(e.To().Type(), e.To().ID()); ok {
		edges := val.(Edges)
		edges.AddEdge(e)
	} else {
		edges := Edges{}
		edges.AddEdge(e)
		g.edgesTo.Set(e.To().Type(), e.To().ID(), edges)
	}
	return nil
}

func (g *graph) HasEdge(id ID) bool {
	_, ok := g.GetEdge(id)
	return ok
}

func (g *graph) GetEdge(id ID) (Edge, bool) {
	val, ok := g.edges.Get(id.Type(), id.ID())
	if ok {
		e, ok := val.(Edge)
		if ok {
			return e, true
		}
	}
	return nil, false
}

func (g *graph) DelEdge(id ID) {
	val, ok := g.edges.Get(id.Type(), id.ID())
	if ok && val != nil {
		edge := val.(Edge)
		fromVal, ok := g.edgesFrom.Get(edge.From().Type(), edge.From().ID())
		if ok && fromVal != nil {
			edges := fromVal.(Edges)
			edges.DelEdge(id)
			g.edgesFrom.Set(edge.From().Type(), edge.From().ID(), edges)
		}
		toVal, ok := g.edgesTo.Get(edge.To().Type(), edge.To().ID())
		if ok && toVal != nil {
			edges := toVal.(Edges)
			edges.DelEdge(id)
			g.edgesTo.Set(edge.To().Type(), edge.To().ID(), edges)
		}
	}
	g.edges.Delete(id.Type(), id.ID())
}

func (g *graph) EdgeTypes() []string {
	return g.edges.Namespaces()
}

func (g *graph) NodeTypes() []string {
	return g.nodes.Namespaces()
}

func (g *graph) AddEdges(e Edges) error {
	var seenErr error
	e.Range(func(edge Edge) bool {
		if err := g.AddEdge(edge); err != nil {
			seenErr = err
			return false
		}
		return true
	})
	return seenErr
}

func (g *graph) DelEdges(e Edges) error {
	var seenErr error
	e.Range(func(edge Edge) bool {
		g.DelEdge(edge)
		return true
	})
	return seenErr
}

func (g *graph) EdgesFrom(id ID) (Edges, bool) {
	val, ok := g.edgesFrom.Get(id.Type(), id.ID())
	if ok {
		if edges, ok := val.(Edges); ok {
			return edges, true
		}
	}
	return nil, false
}

func (g *graph) EdgesTo(id ID) (Edges, bool) {
	val, ok := g.edgesTo.Get(id.Type(), id.ID())
	if ok {
		if edges, ok := val.(Edges); ok {
			return edges, true
		}
	}
	return nil, false
}

func (g *graph) Close() {
	g.nodes.Close()
	g.edgesTo.Close()
	g.edgesFrom.Close()
	g.edges.Close()
}

func (g *graph) NewID(typ string, id string) ID {
	if id == "" {
		id = genID()
	}
	return &identity{
		id:  id,
		typ: typ,
	}
}

func (g *graph) NewEdge(id ID, attributes Map, from, to Node) Edge {
	return &edge{
		Node: g.NewNode(id, attributes),
		from: from,
		to:   to,
	}
}

func (g *graph) NewNode(i ID, attributes Map) Node {
	return &node{
		id:         i,
		attributes: attributes,
	}
}
