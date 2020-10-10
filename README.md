# Machine

     import "github.com/autom8ter/machine"

Machine is a zero dependency runtime for managed goroutines. It is inspired by errgroup.Group with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with context

- global-cancellable goroutines with context (see Cancel)

- goroutines have IDs and optional tags for easy debugging(see Stats)

- publish/subscribe to channels for passing messages between goroutines

- global concurrency safe cache

## Usage

```go
var ErrNoExist = errors.New("machine: does not exit")
```

#### type Cache

```go
type Cache interface {
	// Get get a value by key and an error if one exists
	Get(key string) (interface{}, error)
	// Range executes the given function on the cache. If the function returns false, the iteration stops.
	Range(fn func(k string, val interface{}) bool) error
	// Set sets the key and value in the cache
	Set(key string, val interface{}) error
	// Del deletes the value by key from the map
	Del(key string) error
}
```

Cache is a concurrency safe cache implementation used by Machine. A default
sync.Map implementation is used if one isn't provided via WithCache

#### type Func

```go
type Func func(routine Routine)
```

Func is the function passed into machine.Go. The Routine is passed into this
function at runtime.

#### func  Every

```go
func Every(ticker *time.Ticker, fn Func) Func
```
Every executes the function every time the ticker ticks until the context
cancels

#### func  HourOfDay

```go
func HourOfDay(hourOfDay int, fn Func) Func
```
HourOfDay executes the function on a give hour of the day

#### type GoOpt

```go
type GoOpt func(o *goOpts)
```

GoOpt is a function that configures GoOpts

#### func  WithPID

```go
func WithPID(id int) GoOpt
```
WithPID is a GoOpt that sets/overrides the process ID of the Routine. A random
id is assigned if this option is not used.

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

- global concurrency safe cache

#### func  New

```go
func New(ctx context.Context, options ...Opt) *Machine
```
New Creates a new machine instance with the given root context & options

#### func (*Machine) Cache

```go
func (m *Machine) Cache() Cache
```

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
               },
               "8afa3f85-b8a6-2708-caeb-bac880b5b89b": {
                   "id": "8afa3f85-b8a6-2708-caeb-bac880b5b89b",
                   "start": "2020-10-04T20:00:21.011062-06:00",
                   "duration": 3051375565,
                   "tags": [
                       "subscribe"
                   ],
               },
               "93da5381-0164-4021-04e6-48b6226a1b78": {
                   "id": "93da5381-0164-4021-04e6-48b6226a1b78",
                   "start": "2020-10-04T20:00:21.01107-06:00",
                   "duration": 3051367098,
                   "tags": [
                       "publish"
                   ],
               }
    }

}

#### func (*Machine) Total

```go
func (p *Machine) Total() int
```
Total returns total goroutines that have been executed by the machine

#### func (*Machine) Wait

```go
func (m *Machine) Wait()
```
Wait blocks until all goroutines exit. This MUST be called after all routines
are added via machine.Go in order for a machine instance to work as intended.

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

#### func  WithCache

```go
func WithCache(cache Cache) Opt
```
WithCache sets the in memory, concurrency safe cache. If not set, a default
sync.Map implementation is used.

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
	// PID() is the goroutines unique process id
	PID() int
	// Tags() are the tags associated with the goroutine
	Tags() []string
	// Start is when the goroutine started
	Start() time.Time
	// Duration is the duration since the goroutine started
	Duration() time.Duration
	// Publish publishes the object to the given channel
	Publish(channel string, obj interface{})
	// Subscribe subscribes to a channel and executes the function on every message passed to it. It exits if the goroutines context is cancelled.
	Subscribe(channel string, handler func(obj interface{}))
	// Machine returns the underlying routine's machine instance
	Machine() *Machine
}
```

Routine is an interface representing a goroutine

#### type RoutineStats

```go
type RoutineStats struct {
	PID      int           `json:"pid"`
	Start    time.Time     `json:"start"`
	Duration time.Duration `json:"duration"`
	Tags     []string      `json:"tags"`
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
