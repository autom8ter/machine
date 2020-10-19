package graph

type Graph interface {
	AddNode(n Node)
	GetNode(id Identifier) (Node, bool)
	DelNode(id Identifier)
	HasNode(id Identifier) bool
	NodeTypes() []string
	AddEdge(e Edge)
	GetEdge(id Identifier) (Edge, bool)
	HasEdge(id Identifier) bool
	DelEdge(id Identifier)
	EdgesFrom(id Identifier) (Edges, bool)
	EdgesTo(id Identifier) (Edges, bool)
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

func (g *graph) AddEdge(e Edge) {
	g.edges.Set(e.Type(), e.ID(), e)
	if val, ok := g.edgesFrom.Get(e.From().Type(), e.From().ID()); ok {
		edges := val.(Edges)
		edges[e.Type()] = append(edges[e.Type()], e)
	} else {
		edges := Edges{}
		edges[e.Type()] = []Edge{e}
		g.edgesFrom.Set(e.From().Type(), e.From().ID(), edges)
	}
	if val, ok := g.edgesTo.Get(e.To().Type(), e.To().ID()); ok {
		edges := val.(Edges)
		edges[e.Type()] = append(edges[e.Type()], e)
	} else {
		edges := Edges{}
		edges[e.Type()] = []Edge{e}
		g.edgesTo.Set(e.To().Type(), e.To().ID(), edges)
	}
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
			removeEdge(edges[id.Type()], id)
			g.edgesFrom.Set(edge.From().Type(), edge.From().ID(), edges)
		}
		toVal, ok := g.edgesTo.Get(edge.To().Type(), edge.To().ID())
		if ok && toVal != nil {
			edges := toVal.(Edges)
			removeEdge(edges[id.Type()], id)
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
