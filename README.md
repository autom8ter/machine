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

Machine is a runtime for managed goroutines. It is inspired by errgroup.Group
with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with context

- global-cancellable goroutines with context (see Cancel)

- tagging goroutines for debugging(see Stats)

- publish/subscribe to channels for passing messages between goroutines

#### func  New

```go
func New(ctx context.Context, options ...Opt) (*Machine, error)
```
New Creates a new machine instance with the given root context & options

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
func (m *Machine) Go(fn func(routine Routine) error, tags ...string)
```
Go calls the given function in a new goroutine.

The first call to return a non-nil error who's cause is CancelGroup cancels the
context of every job. All errors that are not CancelGroup will be returned by
Wait.

#### func (*Machine) Stats

```go
func (m *Machine) Stats() Stats
```
Stats returns Goroutine information from the machine

#### func (*Machine) Wait

```go
func (p *Machine) Wait() []error
```
Wait waites for all goroutines to exit

#### type Opt

```go
type Opt func(o *option)
```

Opt is a single option when creating a machine instance with New

#### func  WithDebug

```go
func WithDebug(debug bool) Opt
```
WithDebug enables debug mode

#### func  WithMaxRoutines

```go
func WithMaxRoutines(max int) Opt
```
WithMaxRoutines throttles goroutines at the input number. It will panic if <=
zero.

#### func  WithPublishChannelBuffer

```go
func WithPublishChannelBuffer(length int) Opt
```
WithPublishChannelBuffer sets the buffer length of the channel returned from a
goroutine publishTo

#### func  WithSubscribeChannelBuffer

```go
func WithSubscribeChannelBuffer(length int) Opt
```
WithSubscribeChannelBuffer sets the buffer length of the channel returned from a
Routine subscribeTo

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
	// PublishTo starts a stream that may be published to from the routine. It listens on the returned channel.
	PublishTo(channel string) chan interface{}
	// SubscribeTo subscribes to a channel & returns a go channel
	SubscribeTo(channel string) chan interface{}
	// Subscriptions returns the channels that this goroutine is subscribed to
	Subscriptions() []string
	// AddedAt returns the goroutine count before the goroutine was added
	AddedAt() int
	//Done cancels the context of the current goroutine & kills any of it's subscriptions
	Done()
}
```

Routine is an interface representing a goroutine

#### type RoutineStats

```go
type RoutineStats struct {
	ID            string        `json:"id"`
	Start         time.Time     `json:"start"`
	Duration      time.Duration `json:"duration"`
	Tags          []string      `json:"tags"`
	AddedAt       int           `json:"addedAt"`
	Subscriptions []string      `json:"subscriptions"`
}
```

RoutineStats holds information about a single goroutine

#### type Stats

```go
type Stats struct {
	Count    int                     `json:"count"`
	Routines map[string]RoutineStats `json:"routines"`
}
```

Stats holds information about goroutines

#### func (Stats) String

```go
func (s Stats) String() string
```
String prints a pretty json string of the stats
