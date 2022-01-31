package machine_test

import (
	"context"
	"fmt"
	"github.com/autom8ter/machine/v3"
	"sort"
	"sync"
	"time"
)

func ExampleNew() {
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
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
		return nil
	})
	m.Go(ctx, func(ctx context.Context) error {
		m.Subscribe(ctx, "engineering.*", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
		return nil
	})
	m.Go(ctx, func(ctx context.Context) error {
		m.Subscribe(ctx, "human_resources.*", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
		return nil
	})
	m.Go(ctx, func(ctx context.Context) error {
		m.Subscribe(ctx, "*", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
		return nil
	})
	<-time.After(1 * time.Second)
	m.Publish(ctx, machine.Message{
		Channel: "human_resources.chat_room6",
		Body:    "hello world human resources",
	})
	m.Publish(ctx, machine.Message{
		Channel: "accounting.chat_room2",
		Body:    "hello world accounting",
	})
	m.Publish(ctx, machine.Message{
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
}
