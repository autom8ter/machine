package graph

import "fmt"

// Graph is a concurrency safe directed Graph datastructure
type Graph interface {
	// NewIdentifier is a constructor for an Identifier. If an id is not specified, a random one will be generated automatically.
	NewIdentifier(typ string, id string) Identifier
	// NewNode is a constructor for a graph node
	NewEdge(id Identifier, attributes Map, from, to Node) Edge
	// NewNode is a constructor for a graph node
	NewNode(id Identifier, attributes Map) Node
	// AddNode adds a single node to the graph
	AddNode(n Node)
	// GetNode gets a node from the graph if it exists
	GetNode(id Identifier) (Node, bool)
	// DelNode deletes the nodes and it's edges
	DelNode(id Identifier)
	// HasNode returns true if the node exists in the graph
	HasNode(id Identifier) bool
	// NodeTypes returns an array of Node types that exist in the graph
	NodeTypes() []string
	// AddEdge adds an edge to the graph between the two nodes. It returns an error if one of the nodes does not exist.
	AddEdge(e Edge) error
	// GetEdge gets the edge by id if it exists
	GetEdge(id Identifier) (Edge, bool)
	// HasEdge returns true if the edge exists in the graph
	HasEdge(id Identifier) bool
	// DelEdge deletes the edge
	DelEdge(id Identifier)
	// EdgesFrom returns all of the edges the point from the given node identifier
	EdgesFrom(id Identifier) (Edges, bool)
	// EdgesTo returns all of the edges the point to the given node identifier
	EdgesTo(id Identifier) (Edges, bool)
	// EdgeTypes returns all of the edge types in the Graph
	EdgeTypes() []string
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

func (g *graph) GetNode(id Identifier) (Node, bool) {
	val, ok := g.nodes.Get(id.Type(), id.ID())
	if ok {
		n, ok := val.(Node)
		if ok {
			return n, true
		}
	}
	return nil, false
}

func (g *graph) HasNode(id Identifier) bool {
	_, ok := g.GetNode(id)
	return ok
}

func (g *graph) DelNode(id Identifier) {
	if val, ok := g.edgesFrom.Get(id.Type(), id.ID()); ok {
		if val != nil {
			edges := val.(Edges)
			for _, edgeList := range edges {
				for _, e := range edgeList {
					g.DelEdge(e)
				}
			}
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
		edges[e.Type()][e.ID()] = e
	} else {
		edges := Edges{}
		edges[e.Type()] = map[string]Edge{
			e.ID(): e,
		}
		g.edgesFrom.Set(e.From().Type(), e.From().ID(), edges)
	}
	if val, ok := g.edgesTo.Get(e.To().Type(), e.To().ID()); ok {
		edges := val.(Edges)
		edges[e.Type()][e.ID()] = e
	} else {
		edges := Edges{}
		edges[e.Type()] = map[string]Edge{
			e.ID(): e,
		}
		g.edgesTo.Set(e.To().Type(), e.To().ID(), edges)
	}
	return nil
}

func (g *graph) HasEdge(id Identifier) bool {
	_, ok := g.GetEdge(id)
	return ok
}

func (g *graph) GetEdge(id Identifier) (Edge, bool) {
	val, ok := g.edges.Get(id.Type(), id.ID())
	if ok {
		e, ok := val.(Edge)
		if ok {
			return e, true
		}
	}
	return nil, false
}

func (g *graph) DelEdge(id Identifier) {
	val, ok := g.edges.Get(id.Type(), id.ID())
	if ok && val != nil {
		edge := val.(Edge)
		fromVal, ok := g.edgesFrom.Get(edge.From().Type(), edge.From().ID())
		if ok && fromVal != nil {
			edges := fromVal.(Edges)

			delete(edges[id.Type()], id.ID())
			g.edgesFrom.Set(edge.From().Type(), edge.From().ID(), edges)
		}
		toVal, ok := g.edgesTo.Get(edge.To().Type(), edge.To().ID())
		if ok && toVal != nil {
			edges := toVal.(Edges)
			delete(edges[id.Type()], id.ID())
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

func (g *graph) EdgesFrom(id Identifier) (Edges, bool) {
	val, ok := g.edgesFrom.Get(id.Type(), id.ID())
	if ok {
		if edges, ok := val.(Edges); ok {
			return edges, true
		}
	}
	return nil, false
}

func (g *graph) EdgesTo(id Identifier) (Edges, bool) {
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

func removeEdge(slice []Edge, id Identifier) []Edge {
	for i, e := range slice {
		if e.Type() == id.Type() && e.ID() == id.ID() {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func (g *graph) NewIdentifier(typ string, id string) Identifier {
	if id == "" {
		id = genID()
	}
	return &identity{
		id:  id,
		typ: typ,
	}
}

func (g *graph) NewEdge(id Identifier, attributes Map, from, to Node) Edge {
	return &edge{
		Node: g.NewNode(id, attributes),
		from: from,
		to:   to,
	}
}

func (g *graph) NewNode(id Identifier, attributes Map) Node {
	return &node{
		Identifier: id,
		attributes: attributes,
	}
}
