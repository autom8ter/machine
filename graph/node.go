package graph

// Node is a single element in the Graph. It may be connected to other Nodes via Edges
type Node interface {
	Identifier
	// Attributes returns arbitrary data found in the node(if it exists)
	Attributes() Map
}

type node struct {
	Identifier
	attributes Map
}

func (n *node) Attributes() Map {
	return n.attributes
}

func NewNode(id Identifier, attributes Map) Node {
	return &node{
		Identifier: id,
		attributes: attributes,
	}
}
