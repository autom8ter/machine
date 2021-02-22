# Machine [![GoDoc](https://godoc.org/github.com/autom8ter/machine?status.svg)](https://godoc.org/github.com/autom8ter/machine)

![concurrency](images/concurrency.jpg)


`import "github.com/autom8ter/machine/v2"`

```go
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  	defer cancel()
  	var (
  		m       = machine.New()
  		results []string
  		mu      sync.RWMutex
  	)
  	defer m.Close()
  
  	m.Go(ctx, func(ctx context.Context) error {
  		m.Subscribe(ctx, "accounting.*", func(ctx context.Context, msg machine.Message) (bool, error) {
  			mu.Lock()
  			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.GetChannel(), msg.GetBody()))
  			mu.Unlock()
  			return true, nil
  		})
  		return nil
  	})
  	m.Go(ctx, func(ctx context.Context) error {
  		m.Subscribe(ctx, "engineering.*", func(ctx context.Context, msg machine.Message) (bool, error) {
  			mu.Lock()
  			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.GetChannel(), msg.GetBody()))
  			mu.Unlock()
  			return true, nil
  		})
  		return nil
  	})
  	m.Go(ctx, func(ctx context.Context) error {
  		m.Subscribe(ctx, "human_resources.*", func(ctx context.Context, msg machine.Message) (bool, error) {
  			mu.Lock()
  			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.GetChannel(), msg.GetBody()))
  			mu.Unlock()
  			return true, nil
  		})
  		return nil
  	})
  	m.Go(ctx, func(ctx context.Context) error {
  		m.Subscribe(ctx, "*", func(ctx context.Context, msg machine.Message) (bool, error) {
  			mu.Lock()
  			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.GetChannel(), msg.GetBody()))
  			mu.Unlock()
  			return true, nil
  		})
  		return nil
  	})
  	<-time.After(1 * time.Second)
  	m.Publish(ctx, machine.Msg{
  		Channel: "human_resources.chat_room6",
  		Body:    "hello world human resources",
  	})
  	m.Publish(ctx, machine.Msg{
  		Channel: "accounting.chat_room2",
  		Body:    "hello world accounting",
  	})
  	m.Publish(ctx, machine.Msg{
  		Channel: "engineering.chat_room1",
  		Body:    "hello world engineering",
  	})
  	m.Wait()
  	sort.Strings(results)
  	for _, res := range results {
  		fmt.Print(res)
  	}
  	// Output:
  	//(accounting.chat_room2) received msg: hello world accounting
  	//(accounting.chat_room2) received msg: hello world accounting
  	//(engineering.chat_room1) received msg: hello world engineering
  	//(engineering.chat_room1) received msg: hello world engineering
  	//(human_resources.chat_room6) received msg: hello world human resources
  	//(human_resources.chat_room6) received msg: hello world human resources
```

[Machine](https://pkg.go.dev/github.com/autom8ter/machine/v2#Machine) is a zero dependency library for highly concurrent Go applications. 
It is inspired by [`errgroup`](https://pkg.go.dev/golang.org/x/sync/errgroup)`.`[`Group`](https://pkg.go.dev/golang.org/x/sync/errgroup#Group) with extra bells & whistles:
- In memory Publish Subscribe for asynchronously broadcasting & consuming messages in memory
- Asynchronous worker groups similar to errgroup.Group
- Asynchronous error handling(see `machine.Wait(fn func(err error))`)


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


## Examples (v2)

All examples are < 500 lines of code(excluding code generation)

- [gRPC Bidirectional Chat Stream Server](v2/examples/README.md#grpc-bidirectional-chat-server)

