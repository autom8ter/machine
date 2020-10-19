package graph_test

import (
	"github.com/autom8ter/machine/graph"
	"testing"
)

func Test(t *testing.T) {
	g := graph.NewGraph()
	coleman := g.NewNode(g.NewIdentifier("user", "cword"), graph.Map{
		"job_title": "Software Engineer",
	})
	tyler := g.NewNode(g.NewIdentifier("user", "twash"), graph.Map{
		"job_title": "Carpenter",
	})
	g.AddNode(coleman)
	g.AddNode(tyler)
	colemansBFF := g.NewEdge(g.NewIdentifier("friend", "bff"), graph.Map{
		"source": "school",
	}, coleman, tyler)
	g.AddEdge(colemansBFF)
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
