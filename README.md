# machine
--
    import "github.com/autom8ter/machine"


## Usage

```go
var Cancel = errors.New("[machine] cancel")
```
if a goroutine returns this error, every goroutines context will be cancelled

#### type Func

```go
type Func func(routine Routine) error
```

Func is the function passed into machine.Go. The Routine is passed into this
function at runtime.

#### type GoOpt

```go
type GoOpt func(o *goOpts)
```

GoOpt is a function that configures GoOpts

#### func  WithID

```go
func WithID(id string) GoOpt
```
WithID is a GoOpt that sets/overrides the ID of the Routine. A random uuid is
assigned if this option is not used.

#### func  WithTags

```go
func WithTags(tags ...string) GoOpt
```
WithTags is a GoOpt that adds an array of strings as "tags" to the Routine.

#### func  WithTimeout

```go
func WithTimeout(to time.Duration) GoOpt
```
WithTimeout is a GoOpt that creates the Routine's context with the given timeout
value

#### type Machine

```go
type Machine struct {
}
```

Machine is a zero dependency runtime for managed goroutines. It is inspired by
errgroup.Group with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with context

- global-cancellable goroutines with context (see Cancel)

- goroutines have IDs and optional tags for easy debugging(see Stats)

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
func (m *Machine) Go(fn Func, opts ...GoOpt)
```
Go calls the given function in a new goroutine.

The first call to return a non-nil error who's cause is machine.Cancel cancels
the context of every job. All errors that are not of type machine.Cancel will be
returned by Wait.

#### func (*Machine) Stats

```go
func (m *Machine) Stats() Stats
```
Stats returns Goroutine information from the machine example:

{

           "count": 3,
           "routines": {
               "021851f5-d9ac-0f31-3a89-ddfc454c5f8f": {
                   "id": "021851f5-d9ac-0f31-3a89-ddfc454c5f8f",
                   "start": "2020-10-04T20:00:21.061072-06:00",
                   "duration": 3001366067,
                   "tags": [
                       "stream-to-acme.com"
                   ],
                   "addedAt": 0,
                   "subscriptions": null
               },
               "8afa3f85-b8a6-2708-caeb-bac880b5b89b": {
                   "id": "8afa3f85-b8a6-2708-caeb-bac880b5b89b",
                   "start": "2020-10-04T20:00:21.011062-06:00",
                   "duration": 3051375565,
                   "tags": [
                       "subscribe"
                   ],
                   "addedAt": 0,
                   "subscriptions": [
                       "acme.com"
                   ]
               },
               "93da5381-0164-4021-04e6-48b6226a1b78": {
                   "id": "93da5381-0164-4021-04e6-48b6226a1b78",
                   "start": "2020-10-04T20:00:21.01107-06:00",
                   "duration": 3051367098,
                   "tags": [
                       "publish"
                   ],
                   "addedAt": 0,
                   "subscriptions": null
               }
    }

}

#### func (*Machine) Wait

```go
func (p *Machine) Wait() []error
```
Wait waites for all goroutines to exit

#### type Middleware

```go
type Middleware func(fn Func) Func
```

Middleware is a function that wraps/modifies the behavior of a machine.Func.

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

#### func  WithMiddlewares

```go
func WithMiddlewares(middlewares ...Middleware) Opt
```
WithMiddlewares adds middlewares to the machine that will wrap every machine.Go
Func that is executed by the machine instance.

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
