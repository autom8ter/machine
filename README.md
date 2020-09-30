# machine
--
    import "github.com/autom8ter/machine"


## Usage

```go
var Cancel = errors.New("[machine] cancel")
```
if a goroutine returns this error, every goroutines context will be cancelled

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
New Creates a new machine instance

#### func (*Machine) Cancel

```go
func (p *Machine) Cancel()
```
Cancel cancels every goroutines context

#### func (*Machine) Current

```go
func (p *Machine) Current() int
```
Current returns current managed goroutine count

#### func (*Machine) Go

```go
func (p *Machine) Go(f func(routine Routine) error, tags ...string)
```
Go calls the given function in a new goroutine.

The first call to return a non-nil error who's cause is CancelGroup cancels the
context of every job. All errors that are not CancelGroup will be returned by
Wait.

#### func (*Machine) Stats

```go
func (m *Machine) Stats() Stats
```
Stats returns Goroutine information

#### func (*Machine) Wait

```go
func (p *Machine) Wait() []error
```
Wait waites for all goroutines to exit

#### type Opts

```go
type Opts struct {
	// MaxRoutines throttles goroutines at the given count
	MaxRoutines int
	// Debug enables debug logs
	Debug bool
}
```

Opts are options when creating a machine instance

#### type Routine

```go
type Routine interface {
	// Context returns the goroutines unique context that may be used for cancellation
	Context() context.Context
	// ID() is the goroutines unique id
	ID() string
	// Tags() are the tags associated with the goroutine
	Tags() []string
	// Start is when the goroutine started
	Start() time.Time
	// Duration is the duration since the goroutine started
	Duration() time.Duration
	// Publish publishes the object to the given channel
	Publish(channel string, obj interface{})
	// Subscribe subscribes to a channel & returns a go channel
	Subscribe(channel string) chan interface{}
	// Subscriptions returns the channels that this goroutine is subscribed to
	Subscriptions() []string
}
```

Routine is an interface representing a goroutine

#### type Stats

```go
type Stats struct {
	Count    int
	Routines map[string]Routine
}
```

Stats holds information about goroutines
