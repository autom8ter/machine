# Machine [![GoDoc](https://godoc.org/github.com/autom8ter/machine?status.svg)](https://godoc.org/github.com/autom8ter/machine)

```go
import "github.com/autom8ter/machine"

m := machine.New(context.Background(),
	// functions are added to a FIFO channel that will block when active routines == max routines. 
	machine.WithMaxRoutines(10),
        // every function executed by machine.Go will recover from panics
	machine.WithMiddlewares(machine.PanicRecover()),
	// WithValue passes the value to the root context of the machine- it is available in the context of all child machine's & all Routine's
        machine.WithValue("testing", true),
        // WithTimeout cancels the machine's context after the given timeout
        machine.WithTimeout(30 *time.Second)
)
defer m.Close()

channelName := "acme.com"

// start a goroutine that subscribes to all messages sent to the target channel for 5 seconds
m.Go(func(routine machine.Routine) {
		routine.Subscribe(channelName, func(obj interface{}) {
			fmt.Printf("%v | subscription msg received! channel = %v msg = %v stats = %s\n",
				routine.PID(), channelName, obj, m.Stats().String())
		})
	},
	machine.GoWithTags("subscribe"),
	machine.GoWithTimeout(5*time.Second),
)

// start another goroutine that publishes to the target channel every second for 5 seconds
m.Go(func(routine machine.Routine) {
		fmt.Printf("%v | streaming msg to channel = %v stats = %s\n", routine.PID(), channelName, routine.Machine().Stats().String())
		// publish message to channel
		routine.Publish(channelName, "hey there bud!")
	},
	machine.GoWithTags("publish"),
	machine.GoWithTimeout(5*time.Second),
	machine.GoWithMiddlewares(
		// run every second until context cancels
		machine.Cron(time.NewTicker(1*time.Second)),
	),
)

m.Wait()
```

[Machine](https://pkg.go.dev/github.com/autom8ter/machine#Machine) is a zero dependency library for highly concurrent Go applications. It is inspired by [`errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup)`.`[`Group`](https://pkg.go.dev/golang.org/x/sync/errgroup#Group) with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with [`Context`](https://golang.org/pkg/context#Context)

- global-cancellable goroutines with context (see [Cancel](https://pkg.go.dev/github.com/autom8ter/machine#Machine.Cancel))

- goroutines have IDs and optional tags for easy debugging (see [Stats](https://pkg.go.dev/github.com/autom8ter/machine#Machine.Stats))

- [publish/subscribe](https://pkg.go.dev/github.com/autom8ter/machine#PubSub) to channels for broadcasting messages to active goroutines

- [middlewares](https://pkg.go.dev/github.com/autom8ter/machine#Middleware) for wrapping/decorating functions

- "sub" machines for creating a dependency tree between groups of goroutines

- global, concurrency safe, directed Graph for persisting relational data in memory for use by Routines.

- goroutine leak prevention

## Use Cases

[Machine](https://pkg.go.dev/github.com/autom8ter/machine#Machine) is meant to be completely agnostic and dependency free- its use cases are expected to be emergent.
Really, it can be used **anywhere** goroutines are used. 

Highly concurrent and/or asynchronous applications include:

- gRPC streaming servers

- websocket servers

- pubsub servers

- reverse proxies

- cron jobs

- custom database/cache

- ETL pipelines

- log sink

- filesystem walker

- code generation

## Examples

All examples are < 500 lines of code(excluding code generation)

- [gRPC Bidirectional Chat Stream Server](examples/README.md#grpc-bidirectional-chat-server)
- [TCP Reverse Proxy](examples/README.md#tcp-reverse-proxy)
- [Concurrent Cron Job Server](examples/README.md#concurrent-cron-server)
