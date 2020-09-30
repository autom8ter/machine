package graph

type Metadata interface {
	Get(key string) (interface{}, error)
	Set(key string, val interface{}) error
	Range(fn func(key string, val interface{}) error) error
}


type Node interface {
	ID() uint64
	Type() uint64
	Metadata
	GetEdge() (Edge, error)
}

type Edge interface {
	From() Node
	To() Node
	Node
}

type Graph interface {
	GetNode(typ, id uint64) (Node, error)

}