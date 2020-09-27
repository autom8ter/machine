# machine
--
    import "github.com/autom8ter/machine"


## Usage

```go
var Cancel = errors.New("sync: cancel")
```

#### type Machine

```go
type Machine struct {
}
```

Machine is just like sync.WaitGroup, except it lets you throttle max goroutines.

#### func  New

```go
func New(ctx context.Context, max uint64) *Machine
```

#### func (*Machine) Add

```go
func (p *Machine) Add(delta uint64)
```

#### func (*Machine) AddErr

```go
func (p *Machine) AddErr(err error)
```

#### func (*Machine) Cancel

```go
func (p *Machine) Cancel()
```
Cancel cancels every functions context

#### func (*Machine) Current

```go
func (p *Machine) Current() uint64
```

#### func (*Machine) Done

```go
func (p *Machine) Done()
```

#### func (*Machine) Finished

```go
func (p *Machine) Finished() bool
```

#### func (*Machine) Go

```go
func (p *Machine) Go(f func(ctx context.Context) error)
```
Go calls the given function in a new goroutine.

The first call to return a non-nil error who's cause is CancelGroup cancels the
context of every job. All errors that are not CancelGroup will be returned by
Wait.

#### func (*Machine) Wait

```go
func (p *Machine) Wait() []error
```
