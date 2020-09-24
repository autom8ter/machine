# sync
--
    import "gitlab.com/ftdr/sync"


## Usage

```go
var Cancel = errors.New("sync: cancel")
```

#### type WorkerPool

```go
type WorkerPool struct {
}
```

WorkerPool is just like sync.WaitGroup, except it lets you throttle max
goroutines.

#### func  NewWorkerPool

```go
func NewWorkerPool(ctx context.Context, max int64) *WorkerPool
```

#### func (*WorkerPool) Add

```go
func (p *WorkerPool) Add(delta int64)
```

#### func (*WorkerPool) AddErr

```go
func (p *WorkerPool) AddErr(err error)
```

#### func (*WorkerPool) Cancel

```go
func (p *WorkerPool) Cancel()
```
Cancel cancels every functions context

#### func (*WorkerPool) Current

```go
func (p *WorkerPool) Current() int64
```

#### func (*WorkerPool) Done

```go
func (p *WorkerPool) Done()
```

#### func (*WorkerPool) Go

```go
func (p *WorkerPool) Go(f func(ctx context.Context) error)
```
Go calls the given function in a new goroutine.

The first call to return a non-nil error who's cause is CancelGroup cancels the
context of every job. All errors that are not CancelGroup will be returned by
Wait.

#### func (*WorkerPool) Wait

```go
func (p *WorkerPool) Wait() []error
```
