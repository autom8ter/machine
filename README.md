# Machine [![GoDoc](https://godoc.org/github.com/autom8ter/machine?status.svg)](https://godoc.org/github.com/autom8ter/machine)

     import "github.com/autom8ter/machine"

Machine is a zero dependency runtime for managed goroutines. It is inspired by errgroup.Group with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with context

- global-cancellable goroutines with context (see Cancel)

- goroutines have IDs and optional tags for easy debugging(see Stats)

- publish/subscribe to channels for passing messages between goroutines

- middlewares for wrapping/decorating functions

- "sub" machines for creating a dependency tree between groups of goroutines

- greater than 80% test coverage

## Examples

All examples are < 500 lines of code(excluding code generation)

- [gRPC Bidirectional Chat Stream Server](examples/README.md#grpc-bidirectional-chat-server)
- [TCP Reverse Proxy](examples/README.md#tcp-reverse-proxy)
- [Concurrent Cron Job Server](examples/README.md#concurrent-cron-server)

## Usage

```go
const DefaultMaxRoutines = 1000
```

#### type Func

```go
type Func func(routine Routine)
```

Func is the function passed into machine.Go. The Routine is passed into this
function at runtime.

#### type GoOpt

```go
type GoOpt func(o *goOpts)
```

GoOpt is a function that configures GoOpts

#### func  GoWithMiddlewares

```go
func GoWithMiddlewares(middlewares ...Middleware) GoOpt
```
GoWithMiddlewares wraps the gived function with the input middlewares.

#### func  GoWithPID

```go
func GoWithPID(id int) GoOpt
```
GoWithPID is a GoOpt that sets/overrides the process ID of the Routine. A random
id is assigned if this option is not used.

#### func  GoWithTags

```go
func GoWithTags(tags ...string) GoOpt
```
GoWithTags is a GoOpt that adds an array of strings as "tags" to the Routine.

#### func  GoWithTimeout

```go
func GoWithTimeout(to time.Duration) GoOpt
```
GoWithTimeout is a GoOpt that creates the Routine's context with the given
timeout value

#### type Machine

```go
type Machine struct {
}
```

Machine is a zero dependency runtime for managed goroutines. It is inspired by
errgroup.Group with extra bells & whistles:

#### func  New

```go
func New(ctx context.Context, options ...Opt) *Machine
```
New Creates a new machine instance with the given root context & options

#### func (*Machine) Active

```go
func (p *Machine) Active() int
```
Active returns current active managed goroutine count

#### func (*Machine) Cancel

```go
func (p *Machine) Cancel()
```
Cancel cancels every goroutines context within the machine instance & it's
children

#### func (*Machine) Close

```go
func (m *Machine) Close()
```
Close completely closes the machine instance & all of it's children

#### func (*Machine) Go

```go
func (m *Machine) Go(fn Func, opts ...GoOpt)
```
Go calls the given function in a new goroutine. it is passed information about
the goroutine at runtime via the Routine interface

#### func (*Machine) Parent

```go
func (m *Machine) Parent() *Machine
```
Parent returns the parent Machine instance if it exists and nil if not.

#### func (*Machine) Stats

```go
func (m *Machine) Stats() *Stats
```
Stats returns Goroutine information from the machine

#### func (*Machine) Sub

```go
func (m *Machine) Sub(opts ...Opt) *Machine
```
Sub returns a nested Machine instance that is dependent on the parent machine's
context.

#### func (*Machine) Tags

```go
func (p *Machine) Tags() []string
```
Tags returns the machine's tags

#### func (*Machine) Total

```go
func (p *Machine) Total() int
```
Total returns total goroutines that have been executed by the machine

#### func (*Machine) Wait

```go
func (m *Machine) Wait()
```
Wait blocks until total active goroutine count reaches zero for the instance and
all of it's children. At least one goroutine must have finished in order for
wait to un-block

#### type Middleware

```go
type Middleware func(fn Func) Func
```

Middleware is a function that wraps/modifies the behavior of a machine.Func.

#### func  After

```go
func After(afterFunc func(routine Routine)) Middleware
```
After exectues the afterFunc after the main goroutine exits.

#### func  Before

```go
func Before(beforeFunc func(routine Routine)) Middleware
```
Before exectues the beforeFunc before the main goroutine is executed.

#### func  Cron

```go
func Cron(ticker *time.Ticker) Middleware
```
Cron is a middleware that execute the function every time the ticker ticks until
the goroutine's context cancels

#### func  Decider

```go
func Decider(deciderFunc func(routine Routine) bool) Middleware
```
Decider exectues the deciderFunc before the main goroutine is executed. If it
returns false, the goroutine won't be executed.

#### func  PanicRecover

```go
func PanicRecover() Middleware
```
PanicRecover wraps a goroutine with a middleware the recovers from panics.

#### type Opt

```go
type Opt func(o *option)
```

Opt is a single option when creating a machine instance with New

#### func  WithChildren

```go
func WithChildren(children ...*Machine) Opt
```

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
WithMiddlewares wraps every goroutine function executed by the machine with the
given middlewares. Middlewares can be added to individual goroutines with
GoWithMiddlewares

#### func  WithParent

```go
func WithParent(parent *Machine) Opt
```

#### func  WithPubSub

```go
func WithPubSub(pubsub PubSub) Opt
```
WithPubSub sets the pubsub implementation for the machine instance. An inmemory
implementation is used if none is provided.

#### func  WithTags

```go
func WithTags(tags []string) Opt
```
WithTags sets the machine instances tags

#### type PubSub

```go
type PubSub interface {
	// Publish publishes the object to the channel by name
	Publish(channel string, obj interface{}) error
	// Subscribe subscribes to the given channel
	Subscribe(ctx context.Context, channel string, handler func(obj interface{})) error
	Close()
}
```

PubSub is used to asynchronously pass messages between routines.

#### type Routine

```go
type Routine interface {
	// Context returns the goroutines unique context that may be used for cancellation
	Context() context.Context
	// Cancel cancels the context returned from Context()
	Cancel()
	// PID() is the goroutines unique process id
	PID() int
	// Tags() are the tags associated with the goroutine
	Tags() []string
	// Start is when the goroutine started
	Start() time.Time
	// Duration is the duration since the goroutine started
	Duration() time.Duration
	// Publish publishes the object to the given channel
	Publish(channel string, obj interface{}) error
	// Subscribe subscribes to a channel and executes the function on every message passed to it. It exits if the goroutines context is cancelled.
	Subscribe(channel string, handler func(obj interface{})) error
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
	Tags           []string       `json:"tags"`
	TotalRoutines  int            `json:"totalRoutines"`
	ActiveRoutines int            `json:"activeRoutines"`
	Routines       []RoutineStats `json:"routines"`
	TotalChildren  int            `json:"totalChildren"`
	HasParent      bool           `json:"hasParent"`
}
```

Stats holds information about goroutines

#### func (Stats) String

```go
func (s Stats) String() string
```
String prints a pretty json string of the stats
