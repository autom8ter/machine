package storage

type Record interface {
	ID() uint64
	Type() uint64
	Data() []byte
}
