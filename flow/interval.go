// ref: Allen, Frances E., and John Cocke. "A program data flow analysis
// procedure." Communications of the ACM 19.3 (1976): 137. [1]
//
// [1] https://pdfs.semanticscholar.org/81b9/49a01506a09fcd7ec4faf28e2fa0ec63f1e0.pdf

package flow

import (
	"fmt"

	"github.com/graphism/exp/cfg"

	"gonum.org/v1/gonum/graph"
)

// Intervals returns the intervals contained within the given graph, based on
// the entry node.
func Intervals(g graph.Directed, entry graph.Node) []*Interval {
	var intervals []*Interval
	// 1. Establish a set H for header nodes and initialize it with n_0, the
	// unique entry node for the graph.
	H := newQueue()
	H.push(entry)
	// 2. For h E H, find I(h) as follows:
	for !H.empty() {
		// 5. Select the next unprocessed node in H and repeat steps 2, 3, 4, 5.
		// When there are no more unprocessed nodes in H, the procedure
		// terminates.
		h := H.pop()
		// 2.1. Put h in I(h) as the first element of I(h).
		I := newInterval(g, h)
		for {
			// 2.2. Add to I(h) any node all of whose immediate predecessors are
			// already in I(h).
			n, ok := find2_2(g, entry, I)
			if !ok {
				// 2.3. Repeat 2.2 until no more nodes can be added to I(h).
				break
			}
			I.addNode(n)
		}
		// 3. Add to H all nodes in G which are not already in H and which are not
		// in I(h) but which have immediate predecessors in I(h). Therefore a node
		// is added to H the first time any (but not all) of its immediate
		// predecessors become members of an interval.
		for {
			n, ok := find3(g, entry, I, H)
			if !ok {
				break
			}
			H.push(n)
		}
		intervals = append(intervals, I)
	}
	return intervals
}

func find2_2(g graph.Directed, entry graph.Node, I *Interval) (graph.Node, bool) {
	// 2.2. Add to I(h) any node all of whose immediate predecessors are
	// already in I(h).
loop:
	for _, n := range cfg.SortByRevPost(g.Nodes()) {
		if n == entry {
			continue
		}
		if I.Has(n) {
			// skip if already in I(h).
			continue
		}
		preds := g.To(n)
		// TODO: how to handle nodes without predecessors?
		if len(preds) == 0 {
			panic(fmt.Errorf("invalid node %v; missing predecessors", n))
		}
		for _, pred := range preds {
			if !I.Has(pred) {
				// skip node, as not all immediate predecessors are in I(h).
				continue loop
			}
		}
		return n, true
	}
	return nil, false
}

func find3(g graph.Directed, entry graph.Node, I *Interval, H *queue) (graph.Node, bool) {
	// 3. Add to H all nodes in G which are not already in H and which are not in
	// I(h) but which have immediate predecessors in I(h). Therefore a node is
	// added to H the first time any (but not all) of its immediate predecessors
	// become members of an interval.
	for _, n := range cfg.SortByRevPost(g.Nodes()) {
		if H.has(n) {
			// skip if already in H.
			continue
		}
		if I.Has(n) {
			// skip if already in I(h).
			continue
		}
		preds := g.To(n)
		// TODO: how to handle nodes without predecessors?
		if len(preds) == 0 {
			panic(fmt.Errorf("invalid node %v; missing predecessors", n))
		}
		for _, pred := range preds {
			if I.Has(pred) {
				return n, true
			}
		}
	}
	return nil, false
}

// --- interval

// An Interval I(h) is the maximal, single-entry subgraph in which h is the only
// entry node and in which all closed paths contain h.
type Interval struct {
	// Graph in which the interval exists.
	g graph.Directed
	// Head specifies the entry node of the interval.
	Head graph.Node
	// nodes tracks the nodes contained within the interval.
	nodes map[graph.Node]bool
}

// newInterval returns a new interval with the given header node.
func newInterval(g graph.Directed, head graph.Node) *Interval {
	return &Interval{
		g:    g,
		Head: head,
		nodes: map[graph.Node]bool{
			head: true,
		},
	}
}

// addNode adds the given node to the interval.
func (I *Interval) addNode(n graph.Node) {
	I.nodes[n] = true
}

// Has returns whether the node exists within the interval.
func (I *Interval) Has(n graph.Node) bool {
	return I.nodes[n]
}

// Nodes returns all the nodes in the interval.
func (I *Interval) Nodes() []graph.Node {
	var nodes []graph.Node
	for n := range I.nodes {
		nodes = append(nodes, n)
	}
	return cfg.SortByRevPost(nodes)
}

// [skip start?] embed graph.Directed in Interval, and only implement Has and
// [Nodes methods.

// From returns all nodes that can be reached directly from the given node.
func (I *Interval) From(n graph.Node) []graph.Node {
	return I.g.From(n)
}

// HasEdgeBetween returns whether an edge exists between nodes x and y without
// considering direction.
func (I *Interval) HasEdgeBetween(x, y graph.Node) bool {
	return I.g.HasEdgeBetween(x, y)
}

// Edge returns the edge from u to v if such an edge exists and nil otherwise.
// The node v must be directly reachable from u as defined by the From method.
func (I *Interval) Edge(u, v graph.Node) graph.Edge {
	return I.g.Edge(u, v)
}

// HasEdgeFromTo returns whether an edge exists in the graph from u to v.
func (I *Interval) HasEdgeFromTo(u, v graph.Node) bool {
	return I.g.HasEdgeFromTo(u, v)
}

// To returns all nodes that can reach directly to the given node.
func (I *Interval) To(n graph.Node) []graph.Node {
	return I.g.To(n)
}

// [skip end?]

// --- queue

// A queue is a FIFO queue of nodes.
type queue struct {
	// List of nodes in queue.
	l []graph.Node
	// Current position in queue.
	i int
}

// newQueue returns a new FIFO queue.
func newQueue() *queue {
	return &queue{
		l: make([]graph.Node, 0),
	}
}

// push appends the given node to the end of the queue.
func (q *queue) push(n graph.Node) {
	if !q.has(n) {
		q.l = append(q.l, n)
	}
}

// has reports whether the given node is present in the queue.
func (q *queue) has(n graph.Node) bool {
	for _, m := range q.l {
		if n == m {
			return true
		}
	}
	return false
}

// pop pops and returns the first node of the queue.
func (q *queue) pop() graph.Node {
	if q.empty() {
		panic("invalid call to pop; empty queue")
	}
	n := q.l[q.i]
	q.i++
	return n
}

// empty reports whether the queue is empty.
func (q *queue) empty() bool {
	return len(q.l[q.i:]) == 0
}
