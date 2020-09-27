package graph


type Node interface {
	ID() uint64
	Type() uint64
	Get(key string) (interface{}, error)
	Set(key string, val interface{}) error
	Range(fn func(key string, val interface{}) error) error
}

type Edge interface {
	From() Node
	To() Node
	Node
}

type Graph interface {
	GetNode(typ, id uint64) Node
}