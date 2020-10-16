package machine

import (
	"sync"
)

var routinePool = &pooledRoutines{pool: sync.Pool{New: func() interface{} {
	return new(goRoutine)
}}}

var workPool = &pooledWork{pool: sync.Pool{New: func() interface{} {
	return &work{
		opts: &goOpts{},
		fn:   nil,
	}
}}}

type pooledRoutines struct {
	pool sync.Pool
}

func (p *pooledRoutines) allocateRoutine() *goRoutine {
	return p.pool.Get().(*goRoutine)
}

func (p *pooledRoutines) deallocateRoutine(routine *goRoutine) {
	*routine = goRoutine{}
	p.pool.Put(routine)
}

type pooledWork struct {
	pool sync.Pool
}

func (p *pooledWork) allocateWork() *work {
	return p.pool.Get().(*work)
}

func (p *pooledWork) deallocateWork(w *work) {
	*w.opts = goOpts{}
	w.fn = nil
	p.pool.Put(w)
}