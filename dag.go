package machine

import (
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type DAG interface {
	GetNode(kind string, id string) (*Node, bool)
	AddNode(kind string, node *Node)
	DelNode(kind string, id string)
	Range(kind string, fn func(id string, node *Node) bool)
	HasNode(kind string, id string) bool
	Close()
}

func newDag() DAG {
	return &dag{
		nodes: map[string]*sync.Map{},
		mu:    sync.RWMutex{},
	}
}

type dag struct {
	nodes map[string]*sync.Map
	mu    sync.RWMutex
}

func (d *dag) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, _ := range d.nodes {
		delete(d.nodes, k)
	}
}

func (d *dag) GetNode(kind string, id string) (*Node, bool) {
	d.initKind(kind)
	d.mu.RLock()
	defer d.mu.RUnlock()
	val, ok := d.nodes[kind].Load(id)
	if !ok {
		return nil, false
	}
	return val.(*Node), true
}

func (d *dag) AddNode(kind string, node *Node) {
	d.initKind(kind)
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.nodes[kind].Store(node.id, node)
}

func (d *dag) DelNode(kind string, id string) {
	d.initKind(kind)
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.nodes[kind].Delete(id)
}

func (d *dag) HasNode(kind string, id string) bool {
	d.initKind(kind)
	d.mu.RLock()
	defer d.mu.RUnlock()
	this, ok := d.nodes[kind].Load(id)
	if !ok {
		return false
	}
	if this == nil {
		return false
	}
	return true
}

func (d *dag) Range(kind string, fn func(id string, node *Node) bool) {
	d.initKind(kind)
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.nodes[kind].Range(func(key, value interface{}) bool {
		return fn(key.(string), value.(*Node))
	})
}

func (d *dag) initKind(kind string) {
	if d.nodes[kind] == nil {
		d.mu.Lock()
		d.nodes[kind] = &sync.Map{}
		d.mu.Unlock()
	}
}

type Node struct {
	kind      string
	id        string
	data      *sync.Map
	edges     map[string]map[string]*sync.Map
	edgeMu    sync.RWMutex
	createdAt time.Time
	updatedAt time.Time
	graph     DAG
}

func NewNode(kind string, id string, data map[string]interface{}) *Node {
	if id == "" {
		id = strconv.Itoa(rand.Int())
	}
	d := &sync.Map{}
	for k, v := range data {
		d.Store(k, v)
	}
	return &Node{
		kind:      kind,
		id:        id,
		data:      d,
		edges:     map[string]map[string]*sync.Map{},
		edgeMu:    sync.RWMutex{},
		createdAt: time.Now(),
	}
}

func (n *Node) Kind() string {
	return n.kind
}

func (n *Node) ID() string {
	return n.id
}

func (n *Node) Get(key string) (interface{}, bool) {
	return n.data.Load(key)
}

func (n *Node) Set(key string, val interface{}) {
	n.data.Store(key, val)
	n.updatedAt = time.Now()
}

func (n *Node) Del(key string) {
	n.data.Delete(key)
	n.updatedAt = time.Now()
}

func (n *Node) Range(fn func(key string, val interface{}) bool) {
	n.data.Range(func(key, value interface{}) bool {
		return fn(key.(string), value)
	})
}

func (n *Node) initEdgeKind(edgekind, nodeKind string) {
	if n.edges[edgekind] == nil {
		n.edgeMu.Lock()
		n.edges[edgekind] = map[string]*sync.Map{}
		n.edgeMu.Unlock()
	}
	if n.edges[edgekind][nodeKind] == nil {
		n.edgeMu.Lock()
		n.edges[edgekind][nodeKind] = &sync.Map{}
		n.edgeMu.Unlock()
	}
}

type edge struct {
	cascade bool
}

func (n *Node) AddEdge(edgeKind, nodeKind string, id string, cascade bool) {
	n.initEdgeKind(edgeKind, nodeKind)
	n.edgeMu.RLock()
	defer n.edgeMu.RUnlock()
	n.edges[edgeKind][nodeKind].Store(id, &edge{cascade: cascade})
}

func (n *Node) DelEdge(edgeKind, nodeKind string, id string) {
	n.initEdgeKind(edgeKind, nodeKind)
	n.edgeMu.RLock()
	defer n.edgeMu.RUnlock()
	n.edges[edgeKind][nodeKind].Delete(id)
}

func (n *Node) GetEdge(edgeKind string, nodeKind string, id string) (*Node, bool) {
	n.initEdgeKind(edgeKind, nodeKind)
	n.edgeMu.RLock()
	defer n.edgeMu.RUnlock()
	_, ok := n.edges[edgeKind][nodeKind].Load(id)
	if !ok {
		return nil, false
	}
	return n.graph.GetNode(nodeKind, id)
}

func (n *Node) HasEdge(edgeKind, nodeKind string, id string) bool {
	n.initEdgeKind(edgeKind, nodeKind)
	n.edgeMu.RLock()
	defer n.edgeMu.RUnlock()
	_, ok := n.edges[edgeKind][nodeKind].Load(id)
	if !ok {
		return false
	}
	return true
}

func (n *Node) RangeEdges(edgeKind, nodeKind string, fn func(node *Node) bool) {
	n.initEdgeKind(edgeKind, nodeKind)
	n.edgeMu.RLock()
	defer n.edgeMu.RUnlock()
	n.edges[edgeKind][nodeKind].Range(func(key, value interface{}) bool {
		node, ok := n.graph.GetNode(nodeKind, key.(string))
		if !ok {
			return true
		}
		return fn(node)
	})
}
