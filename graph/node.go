package graph

// Node is a single element in the Graph. It may be connected to other Nodes via Edges
type Node interface {
	ID
	// Attributes returns arbitrary data found in the node(if it exists)
	Attributes() Map
}

func BasicNode(id ID, attributes Map) Node {
	return &node{
		id:         id,
		attributes: attributes,
	}
}

type node struct {
	id         ID
	attributes Map
}

func (n *node) ID() string {
	return n.id.ID()
}

func (n *node) Type() string {
	return n.id.Type()
}

func (n *node) String() string {
	return n.id.String()
}

func (n *node) Attributes() Map {
	return n.attributes
}
