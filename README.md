# machine
--
    import "github.com/autom8ter/machine"


## Usage

```go
var Cancel = errors.New("[machine] cancel")
```

#### type Machine

```go
type Machine struct {
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
func (p *Machine) Current() int
```
Current returns current managed goroutine count

#### func (*Machine) Go

```go
func (p *Machine) Go(f func(ctx context.Context) error, tags ...string)
```
Go calls the given function in a new goroutine.

The first call to return a non-nil error who's cause is CancelGroup cancels the
context of every job. All errors that are not CancelGroup will be returned by
Wait.

#### func (*Machine) Stats

```go
func (m *Machine) Stats() Stats
```

#### func (*Machine) Wait

```go
func (p *Machine) Wait() []error
```

#### type Opts

```go
type Opts struct {
	MaxRoutines int
	Debug       bool
}
```


#### type Routine

```go
type Routine struct {
	ID       string
	Tags     []string
	Start    time.Time
	Duration time.Duration
}
```


#### type Stats

```go
type Stats struct {
	Count    int
	Routines map[string]*Routine
}
```
