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
}
