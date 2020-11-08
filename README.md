# Machine [![GoDoc](https://godoc.org/github.com/autom8ter/machine?status.svg)](https://godoc.org/github.com/autom8ter/machine)

`import "github.com/autom8ter/machine"`

```go
        ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := machine.New(ctx,
		machine.WithMaxRoutines(10),
		machine.WithMiddlewares(machine.PanicRecover()),
	)
	defer m.Close()

	channelName := "acme.com"
	const publisherID = "publisher"
	// start another goroutine that publishes to the target channel every second for 5 seconds OR the routine's context cancels
	m.Go(func(routine machine.Routine) {
		fmt.Printf("%v | streaming msg to channel = %v stats = %s\n", routine.PID(), channelName, routine.Machine().Stats().String())
		// publish message to channel
		routine.Publish(channelName, "hey there bud!")
	}, machine.GoWithTags("publish"),
		machine.GoWithPID(publisherID),
		machine.GoWithTimeout(5*time.Second),
		machine.GoWithMiddlewares(
			// run every second until context cancels
			machine.Cron(time.NewTicker(1*time.Second)),
		),
	)
	// start a goroutine that subscribes to all messages sent to the target channel for 3 seconds OR the routine's context cancels
	m.Go(func(routine machine.Routine) {
		routine.Subscribe(channelName, func(obj interface{}) {
			fmt.Printf("%v | subscription msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
		})
	}, machine.GoWithTags("subscribe"),
		machine.GoWithTimeout(3*time.Second),
	)

	// start a goroutine that subscribes to just the first two messages it receives on the channel OR the routine's context cancels
	m.Go(func(routine machine.Routine) {
		routine.SubscribeN(channelName, 2, func(obj interface{}) {
			fmt.Printf("%v | subscription msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
		})
	}, machine.GoWithTags("subscribeN"))

	// check if the machine has the publishing routine
	exitAfterPublisher := func() bool {
		return m.HasRoutine(publisherID)
	}

	// start a goroutine that subscribes to the channel until the publishing goroutine exits OR the routine's context cancels
	m.Go(func(routine machine.Routine) {
		routine.SubscribeUntil(channelName, exitAfterPublisher, func(obj interface{}) {
			fmt.Printf("%v | subscription msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
		})
	}, machine.GoWithTags("subscribeUntil"))

	m.Wait()
```

[Machine](https://pkg.go.dev/github.com/autom8ter/machine#Machine) is a zero dependency library for highly concurrent Go applications. It is inspired by [`errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup)`.`[`Group`](https://pkg.go.dev/golang.org/x/sync/errgroup#Group) with extra bells & whistles:

- throttled goroutines

- self-cancellable goroutines with [`Context`](https://golang.org/pkg/context#Context)

- global-cancellable goroutines with context (see [Cancel](https://pkg.go.dev/github.com/autom8ter/machine#Machine.Cancel))

- goroutines have IDs and optional tags for easy debugging (see [Stats](https://pkg.go.dev/github.com/autom8ter/machine#Machine.Stats))

- native [publish/subscribe](https://pkg.go.dev/github.com/autom8ter/machine/pubsub#PubSub) implementation for broadcasting messages to active goroutines

- native global, concurrency safe, directed [graph](https://pkg.go.dev/github.com/autom8ter/machine/graph#Graph) implementation for persisting relational data in memory for use by Routines.

- [middlewares](https://pkg.go.dev/github.com/autom8ter/machine#Middleware) for wrapping/decorating functions

- "sub" machines for creating a dependency tree between groups of goroutines

- goroutine leak prevention

- native pprof & golang execution tracer integration

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

