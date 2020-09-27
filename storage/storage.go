package storage

type Storage interface {
	Get(key uint64) Record
	Set()

}
