# machine
--
    import "github.com/autom8ter/machine"


## Usage

```go
var Cancel = errors.New("machine: cancel")
```

#### type Machine

```go
type Machine struct {
	Storage
}
```

Machine is just like sync.WaitGroup, except it lets you throttle max goroutines.

#### func  New

```go
func New(ctx context.Context, opts *Opts) (*Machine, error)
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

#### type Opts

```go
type Opts struct {
	MaxRoutines     uint64
	StorageProvider Storage
	SyncInterval    time.Duration
}
```


#### type Storage

```go
type Storage interface {
	Get(id string) (map[string]interface{}, error)
	Set(id string, data map[string]interface{}) error
	Range(fn func(key string, data map[string]interface{}) bool)
	Sync() error
	Del(id string) error
	Close() error
}
```


#### func  NewInMemStorage

```go
func NewInMemStorage() Storage
```
