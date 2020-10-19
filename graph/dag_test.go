package graph_test

import (
	"fmt"
	"github.com/autom8ter/machine/graph"
	"testing"
	"time"
)

func Test(t *testing.T) {
	g := graph.NewGraph()
	coleman := g.NewNode(g.NewID("user", "cword"), graph.Map{
		"job_title": "Software Engineer",
	})
	tyler := g.NewNode(g.NewID("user", "twash"), graph.Map{
		"job_title": "Carpenter",
	})
	g.AddNode(coleman)
	g.AddNode(tyler)
	colemansBFF := g.NewEdge(g.NewID("friend", "bff"), graph.Map{
		"source": "school",
	}, coleman, tyler)
	if err := g.AddEdge(colemansBFF); err != nil {
		t.Fatal(err)
	}
	fromColeman, ok := g.EdgesFrom(coleman)
	if !ok {
		t.Fatal("expected at least one edge")
	}
	for _, edgeList := range fromColeman {
		for _, e := range edgeList {
			t.Logf("edge from (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
		}
	}
	toTyler, ok := g.EdgesTo(tyler)
	if !ok {
		t.Fatal("expected at least one edge")
	}
	for _, edgeList := range toTyler {
		for _, e := range edgeList {
			t.Logf("edge to (%s) (%s) -> (%s)", e.String(), e.From().String(), e.To().String())
		}
	}
	g.DelEdge(colemansBFF)
	fromColeman, _ = g.EdgesFrom(coleman)
	if len(fromColeman["friend"]) > 0 {
		t.Fatal("expected zero friend edges")
	}
}

func Benchmark(b *testing.B) {
	b.ReportAllocs()
	g := graph.NewGraph()
	defer g.Close()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		nodeType := fmt.Sprintf("user.%v", time.Now().UnixNano())
		edgeType := fmt.Sprintf("friend.%v", time.Now().UnixNano())
		coleman := g.NewNode(g.NewID(nodeType, "cword"), graph.Map{
			"job_title": "Software Engineer",
		})
		tyler := g.NewNode(g.NewID(nodeType, "twash"), graph.Map{
			"job_title": "Carpenter",
		})
		g.AddNode(coleman)
		g.AddNode(tyler)
		colemansBFF := g.NewEdge(g.NewID(edgeType, ""), graph.Map{
			"source": "school",
		}, coleman, tyler)
		if err := g.AddEdge(colemansBFF); err != nil {
			b.Fatal(err)
		}
		fromColeman, ok := g.EdgesFrom(coleman)
		if !ok {
			b.Fatal("expected at least one edge")
		}
		if fromColeman.Len(edgeType) != 1 {
			b.Fatal("expected one friend")
		}
		toTyler, ok := g.EdgesTo(tyler)
		if !ok {
			b.Fatal("expected at least one edge")
		}
		if toTyler.Len(edgeType) != 1 {
			b.Fatal("expected one friend")
		}
		g.DelEdge(colemansBFF)
		fromColeman, _ = g.EdgesFrom(coleman)
		if len(fromColeman[edgeType]) > 0 {
			b.Fatal("expected zero friend edges")
		}
		g.DelNode(coleman)
		g.DelNode(tyler)
	}
	for _, t := range g.EdgeTypes() {
		b.Logf("edge type = %v\n", t)
	}
}
