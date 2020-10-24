package graph

import (
	"fmt"
	"github.com/autom8ter/machine/cache"
	"github.com/autom8ter/machine/primitive"
)

type Graph struct {
	nodes     *cache.Cache
	edges     *cache.Cache
	edgesFrom *cache.Cache
	edgesTo   *cache.Cache
}

func NewGraph() *Graph {
	return &Graph{
		nodes:     cache.NewCache(),
		edges:     cache.NewCache(),
		edgesFrom: cache.NewCache(),
		edgesTo:   cache.NewCache(),
	}
}

func (g *Graph) EdgeTypes() []string {
	return g.edges.Namespaces()
}

func (g *Graph) NodeTypes() []string {
	return g.nodes.Namespaces()
}

func (g *Graph) AddNode(n *primitive.Node) {
	g.nodes.Set(n.Type(), n.ID(), n)
}

func (g *Graph) AddNodes(nodes ...*primitive.Node) {
	for _, n := range nodes {
		g.AddNode(n)
	}
}

func (g *Graph) GetNode(id primitive.TypedID) (*primitive.Node, bool) {
	val, ok := g.nodes.Get(id.Type(), id.ID())
	if ok {
		n, ok := val.(*primitive.Node)
		if ok {
			return n, true
		}
	}
	return nil, false
}

func (g *Graph) RangeNodeTypes(typ primitive.Type, fn func(n *primitive.Node) bool) {
	g.nodes.Range(typ.Type(), func(key string, val interface{}) bool {
		n, ok := val.(*primitive.Node)
		if ok {
			if !fn(n) {
				return false
			}
		}
		return true
	})
}

func (g *Graph) RangeNodes(fn func(n *primitive.Node) bool) {
	for _, namespace := range g.nodes.Namespaces() {
		g.nodes.Range(namespace, func(key string, val interface{}) bool {
			n, ok := val.(*primitive.Node)
			if ok {
				if !fn(n) {
					return false
				}
			}
			return true
		})
	}
}

func (g *Graph) RangeMap(fn func(e *primitive.Edge) bool) {
	for _, namespace := range g.edges.Namespaces() {
		g.edges.Range(namespace, func(key string, val interface{}) bool {
			e, ok := val.(*primitive.Edge)
			if ok {
				if !fn(e) {
					return false
				}
			}
			return true
		})
	}
}

func (g *Graph) RangeEdgeTypes(edgeType primitive.Type, fn func(e *primitive.Edge) bool) {
	g.edges.Range(edgeType.Type(), func(key string, val interface{}) bool {
		e, ok := val.(*primitive.Edge)
		if ok {
			if !fn(e) {
				return false
			}
		}
		return true
	})
}

func (g *Graph) HasNode(id primitive.TypedID) bool {
	_, ok := g.GetNode(id)
	return ok
}

func (g *Graph) DelNode(id primitive.TypedID) {
	if val, ok := g.edgesFrom.Get(id.Type(), id.ID()); ok {
		if val != nil {
			edges := val.(edgeMap)
			edges.Range(func(e *primitive.Edge) bool {
				g.DelEdge(e)
				return true
			})
		}
	}
	g.nodes.Delete(id.Type(), id.ID())
}

func (g *Graph) AddEdge(e *primitive.Edge) error {
	if !g.HasNode(e.From) {
		return fmt.Errorf("node %s does not exist", e.From.Type())
	}
	if !g.HasNode(e.To) {
		return fmt.Errorf("node does not exist")
	}
	g.edges.Set(e.Type(), e.ID(), e)
	if val, ok := g.edgesFrom.Get(e.From.Type(), e.From.ID()); ok {
		edges := val.(edgeMap)
		edges.AddEdge(e)
	} else {
		edges := edgeMap{}
		edges.AddEdge(e)
		g.edgesFrom.Set(e.From.Type(), e.From.ID(), edges)
	}
	if val, ok := g.edgesTo.Get(e.To.Type(), e.To.ID()); ok {
		edges := val.(edgeMap)
		edges.AddEdge(e)
	} else {
		edges := edgeMap{}
		edges.AddEdge(e)
		g.edgesTo.Set(e.To.Type(), e.To.ID(), edges)
	}
	return nil
}

func (g *Graph) AddEdges(edges ...*primitive.Edge) error {
	for _, e := range edges {
		if err := g.AddEdge(e); err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) HasEdge(id primitive.TypedID) bool {
	_, ok := g.GetEdge(id)
	return ok
}

func (g *Graph) GetEdge(id primitive.TypedID) (*primitive.Edge, bool) {
	val, ok := g.edges.Get(id.Type(), id.ID())
	if ok {
		e, ok := val.(*primitive.Edge)
		if ok {
			return e, true
		}
	}
	return nil, false
}

func (g *Graph) DelEdge(id primitive.TypedID) {
	val, ok := g.edges.Get(id.Type(), id.ID())
	if ok && val != nil {
		edge := val.(*primitive.Edge)
		fromVal, ok := g.edgesFrom.Get(edge.From.Type(), edge.From.ID())
		if ok && fromVal != nil {
			edges := fromVal.(edgeMap)
			edges.DelEdge(id)
			g.edgesFrom.Set(edge.From.Type(), edge.From.ID(), edges)
		}
		toVal, ok := g.edgesTo.Get(edge.To.Type(), edge.To.ID())
		if ok && toVal != nil {
			edges := toVal.(edgeMap)
			edges.DelEdge(id)
			g.edgesTo.Set(edge.To.Type(), edge.To.ID(), edges)
		}
	}
	g.edges.Delete(id.Type(), id.ID())
}

func (g *Graph) EdgesFrom(id primitive.TypedID, fn func(e *primitive.Edge) bool) {
	val, ok := g.edgesFrom.Get(id.Type(), id.ID())
	if ok {
		if edges, ok := val.(edgeMap); ok {
			edges.Range(func(e *primitive.Edge) bool {
				return fn(e)
			})
		}
	}
}

func (g *Graph) EdgesTo(id primitive.TypedID, fn func(e *primitive.Edge) bool) {
	val, ok := g.edgesTo.Get(id.Type(), id.ID())
	if ok {
		if edges, ok := val.(edgeMap); ok {
			edges.Range(func(e *primitive.Edge) bool {
				return fn(e)
			})
		}
	}
}

func (g *Graph) Close() {
	g.nodes.Close()
	g.edgesTo.Close()
	g.edgesFrom.Close()
	g.edges.Close()
}
