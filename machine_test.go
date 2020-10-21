package machine

import (
	"context"
	"fmt"
	"github.com/autom8ter/machine/graph"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

func Test(t *testing.T) {
	t.Run("e2e", runE2ETest)
	t.Run("stats", runStatsTest)
	t.Run("graph", runGraphTest)
}

func runE2ETest(t *testing.T) {
	cpu, err := os.Create("testing.cpu.prof")
	if err != nil {
		t.Fatal(err)
	}
	cpu.Truncate(0)
	defer cpu.Close()
	pprof.StartCPUProfile(cpu)
	defer pprof.StopCPUProfile()
	m := New(context.Background(),
		WithMaxRoutines(10),
		WithMiddlewares(PanicRecover()),
		WithValue("testing", true),
		WithDeadline(time.Now().Add(5*time.Second)),
		WithTags("root"),
	)
	defer m.Close()
	channelName := "acme.com"
	var seen = false
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		msg := "hey there bud!"

		t.Logf("streaming msg to channel = %v msg = %v stats= %s\n", channelName, msg, routine.Machine().Stats().String())
		if err := routine.Publish(channelName, msg); err != nil {
			t.Fatal(err)
		}
	},
		GoWithTags("publish"),
		GoWithPID("publisher"),
		GoWithTimeout(5*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		if err := routine.Subscribe(channelName, func(obj interface{}) {
			seen = true

			t.Logf("subscription msg received! channel = %v msg = %v stats= %s\n", channelName, obj, m.Stats().String())
		}); err != nil {
			t.Fatal(err)
		}
	}, GoWithTags("subscribe"))
	// start a goroutine that subscribes to just the first two messages it receives on the channel
	m.Go(func(routine Routine) {
		routine.SubscribeN(channelName, 2, func(obj interface{}) {
			fmt.Printf("%v | subscriptionN msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
		})
	}, GoWithTags("subscribeN"),
		GoWithTimeout(5*time.Second),
	)
	exitAfterPublisher := func() bool {
		return m.HasRoutine("publisher")
	}
	// start a goroutine that subscribes to just the channel until the publishing goroutine exits
	m.Go(func(routine Routine) {
		routine.SubscribeUntil(channelName, exitAfterPublisher, func(obj interface{}) {
			fmt.Printf("%v | subscriptionUntil msg received! channel = %v msg = %v stats = %s\n", routine.PID(), channelName, obj, m.Stats().String())
		})
	}, GoWithTags("subscribeUntil"),
		GoWithTimeout(5*time.Second),
	)

	m2 := m.Sub(WithMaxRoutines(3))
	defer m2.Close()

	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}

		t.Logf("cron1 stats= %s\n", routine.Machine().Stats().String())
	},
		GoWithTags("cron1"),
		GoWithTimeout(3*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}

		t.Logf("cron2 stats= %s\n", routine.Machine().Stats().String())
	},
		GoWithTags("cron2"),
		GoWithTimeout(3*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Go(func(routine Routine) {
		if routine.Context().Value("testing").(bool) != true {
			t.Fatal("expected testing = true in context")
		}
		t.Logf("cron3 stats= %s\n", routine.Machine().Stats().String())
	},
		GoWithTags("cron3"),
		GoWithTimeout(3*time.Second),
		GoWithMiddlewares(
			Cron(time.NewTicker(1*time.Second)),
		),
	)
	m2.Wait()
	t.Logf("here")
	m.Go(func(routine Routine) {
		panic("panic!")
	})
	m.Wait()
	if m.Active() != 0 {
		t.Fatalf("expected active to be zero, got: %v", m.Active())
	}
	if !seen {
		t.Fatalf("expected to have received subscription msg")
	}
	t.Logf("total = %v", m.Total())
}

func runStatsTest(t *testing.T) {
	m := New(
		context.Background(),
		WithTimeout(3*time.Second),
	)
	defer m.Close()
	for x := 0; x < 100; x++ {
		m.Go(func(routine Routine) {
			fmt.Printf("cron pid = %v\n", routine.PID())
		},
			GoWithMiddlewares(Cron(time.NewTicker(100*time.Millisecond))),
			GoWithTags(fmt.Sprintf("x = %v", x)),
		)
	}
	time.Sleep(1 * time.Second)
	stats := m.Stats()
	if stats.ActiveRoutines != 100 {
		t.Fatalf("expected 100 active routines, got: %v\n", stats.ActiveRoutines)
	}
	m.Wait()

	total := m.Stats().TotalRoutines
	if total != 100 {
		t.Fatalf("expected 100 total routines, got: %v\n", total)
	}
}

func runGraphTest(t *testing.T) {
	m := New(context.Background())
	defer m.Close()
	coleman := graph.BasicNode(graph.BasicID("user", "cword"), graph.Map{
		"job_title": "Software Engineer",
	})
	tyler := graph.BasicNode(graph.BasicID("user", "twash"), graph.Map{
		"job_title": "Carpenter",
	})
	m.Graph().AddNode(coleman)
	m.Graph().AddNode(tyler)
	colemansBFF := graph.BasicEdge(graph.BasicID("friend", "bff"), graph.Map{
		"source": "school",
	}, coleman, tyler)
	m.Graph().AddEdge(colemansBFF)
	fromColeman, ok := m.Graph().EdgesFrom(coleman)
	if !ok {
		t.Fatal("expected at least one edge")
	}
	for _, edgeList := range fromColeman {
		for _, e := range edgeList {
			t.Logf("edge from (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
		}
	}
	toTyler, ok := m.Graph().EdgesTo(tyler)
	if !ok {
		t.Fatal("expected at least one edge")
	}
	for _, edgeList := range toTyler {
		for _, e := range edgeList {
			t.Logf("edge to (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
		}
	}
	m.Graph().DelEdge(colemansBFF)
	fromColeman, _ = m.Graph().EdgesFrom(coleman)
	for _, edgeList := range fromColeman {
		for _, e := range edgeList {
			t.Logf("edge from (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
		}
	}
}

func BenchmarkGo(b *testing.B) {
	b.ReportAllocs()
	m := New(context.Background(), WithMaxRoutines(100))
	defer m.Close()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		m.Go(func(routine Routine) {
			//time.Sleep(100 *time.Millisecond)
			routine.TraceLog("here")
			return
		})
	}
	m.Wait()
}

func BenchmarkSetNode(b *testing.B) {
	b.ReportAllocs()
	m := New(context.Background(), WithTimeout(5*time.Second))
	defer m.Close()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		coleman := graph.BasicNode(graph.BasicID("user", "cword"), graph.Map{
			"job_title": "Software Engineer",
		})
		tyler := graph.BasicNode(graph.BasicID("user", "twash"), graph.Map{
			"job_title": "Carpenter",
		})
		m.Graph().AddNode(coleman)
		m.Graph().AddNode(tyler)
		colemansBFF := graph.BasicEdge(graph.BasicID("friend", ""), graph.Map{
			"source": "school",
		}, coleman, tyler)
		if err := m.Graph().AddEdge(colemansBFF); err != nil {
			b.Fatal(err)
		}
		fromColeman, ok := m.Graph().EdgesFrom(coleman)
		if !ok {
			b.Fatal("expected at least one edge")
		}
		for _, edgeList := range fromColeman {
			for _, e := range edgeList {
				b.Logf("edge from (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
			}
		}
		toTyler, ok := m.Graph().EdgesTo(tyler)
		if !ok {
			b.Fatal("expected at least one edge")
		}
		for _, edgeList := range toTyler {
			for _, e := range edgeList {
				b.Logf("edge to (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
			}
		}
	}
}
