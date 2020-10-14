package machine_test

import (
	"context"
	"fmt"
	"github.com/autom8ter/machine"
	"time"
)

func ExampleNew() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := machine.New(ctx,
		machine.WithMaxRoutines(10),
		machine.WithMiddlewares(machine.PanicRecover()),
	)
	defer m.Close()

	channelName := "acme.com"

	// start a goroutine that subscribes to all messages sent to the target channel for 5 seconds
	m.Go(func(routine machine.Routine) {
		routine.Subscribe(channelName, func(obj interface{}) {
			fmt.Printf("%v | subscription msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
		})
	}, machine.GoWithTags("subscribe"),
		machine.GoWithTimeout(5*time.Second),
	)

	// start another goroutine that publishes to the target channel every second for 5 seconds
	m.Go(func(routine machine.Routine) {
		fmt.Printf("%v | streaming msg to channel = %v stats = %s\n", routine.PID(), channelName, routine.Machine().Stats().String())
		// publish message to channel
		routine.Publish(channelName, "hey there bud!")
	}, machine.GoWithTags("publish"),
		machine.GoWithTimeout(5*time.Second),
		machine.GoWithMiddlewares(
			// run every second until context cancels
			machine.Cron(time.NewTicker(1*time.Second)),
		),
	)

	m.Wait()
}
