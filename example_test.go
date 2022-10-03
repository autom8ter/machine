package machine_test

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/autom8ter/machine/v4"
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
		return m.Subscribe(ctx, "accounting.chat_room2", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
	})
	m.Go(ctx, func(ctx context.Context) error {
		return m.Subscribe(ctx, "engineering.chat_room1", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
	})
	m.Go(ctx, func(ctx context.Context) error {
		return m.Subscribe(ctx, "human_resources.chat_room6", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
	})
	m.Go(ctx, func(ctx context.Context) error {
		return m.Subscribe(ctx, "*", func(ctx context.Context, msg machine.Message) (bool, error) {
			mu.Lock()
			results = append(results, fmt.Sprintf("(%s) received msg: %v\n", msg.Channel, msg.Body))
			mu.Unlock()
			return true, nil
		})
	})
	<-time.After(1 * time.Second)
	m.Publish(ctx, machine.Message{
		Channel: "human_resources.chat_room6",
		Body:    "sending message to human resources",
	})
	m.Publish(ctx, machine.Message{
		Channel: "accounting.chat_room2",
		Body:    "sending message to accounting",
	})
	m.Publish(ctx, machine.Message{
		Channel: "engineering.chat_room1",
		Body:    "sending message to engineering",
	})
	if err := m.Wait(); err != nil {
		panic(err)
	}
	sort.Strings(results)
	for _, res := range results {
		fmt.Print(res)
	}
	// Output:
	//(accounting.chat_room2) received msg: sending message to accounting
	//(accounting.chat_room2) received msg: sending message to accounting
	//(engineering.chat_room1) received msg: sending message to engineering
	//(engineering.chat_room1) received msg: sending message to engineering
	//(human_resources.chat_room6) received msg: sending message to human resources
	//(human_resources.chat_room6) received msg: sending message to human resources
}
