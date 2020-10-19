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

func NewEdge(id Identifier, attributes Map, from, to Node) Edge {
	return &edge{
		Node: NewNode(id, attributes),
		from: from,
		to:   to,
	}
}

type Edges map[string][]Edge
