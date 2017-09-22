package flow

import (
	"fmt"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/traverse"
)

// A Graph represents a control flow graph; a directed graph in which every node
// in the graph is reachable by a path from the entry node.
type Graph interface {
	graph.Directed
	// Entry returns the entry node of the control flow graph.
	Entry() graph.Node
}

// A cfg represents a control flow graph; a directed graph in which every node
// in the graph is reachable by a path from the entry node.
type cfg struct {
	graph.Directed
	// Entry node of the control flow graph.
	entry graph.Node
}

// NewGraph returns a new control flow graph based on the given entry node.
//
// It validates that every node in the graph is reachable by a path from the
// entry node, and panics otherwise.
func NewGraph(g graph.Directed, entry graph.Node) Graph {
	// Check that every node in the graph is reachable by a path from the entry
	// node.
	df := &traverse.DepthFirst{}
	df.Walk(g, entry, nil)
	for _, n := range g.Nodes() {
		if !df.Visited(n) {
			panic(fmt.Errorf("invalid control flow graph; node %v not reachable from entry node %v", n, entry))
		}
	}
	return &cfg{
		Directed: g,
		entry:    entry,
	}
}

// Entry returns the entry node of the control flow graph.
func (g *cfg) Entry() graph.Node {
	return g.entry
}
