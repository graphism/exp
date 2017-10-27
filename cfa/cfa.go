// ref: Cifuentes, Cristina. "Structuring decompiled graphs." Compiler
// Construction. Springer Berlin/Heidelberg, 1996 [1].
//
// [1]: https://pdfs.semanticscholar.org/48bf/d31773af7b67f9d1b003b8b8ac889f08271f.pdf

package cfa

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/graphism/exp/cfg"
	"github.com/graphism/exp/flow"
	"github.com/kr/pretty"
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/path"
)

func Structure(g *cfg.Graph) {
	cfg.InitDFSOrder(g)
	//structLoop(g)
	struct2Way(g)

}

// DerivedGraphSeq returns the derived sequence of graphs, G^1 ... G^n, based on
// the intervals of G.
//
// The first order graph, G^1 is G. The second order graph, G^2, is derived from
// G^1 by collapsing each interval in G^1 into a node. The immediate
// predecessors of the collapsed node are the immediate predecessors of the
// original header node which are not part of the interval. The immediate
// successors are all the immediate, non-interval successors of the original
// exit nodes. Intervals for G^2 are found and the process is repeated until a
// limit flow graph G^n is found. G^n has the property of being a single node or
// an irreducible graph.
func DerivedGraphSeq(src *cfg.Graph) []*cfg.Graph {
	var Gs []*cfg.Graph
	// The first order graph, G^1, is G.
	G := src
	G.SetDOTID("G1")
	createGraph(G)
	Gs = append(Gs, G)
	intNum := 1
	for i := 2; len(G.Nodes()) > 1; i++ {
		Is := flow.Intervals(G, G.Entry())
		for _, I := range Is {
			// Collapse interval into a single node.
			newName := fmt.Sprintf("I%d", intNum)
			delNodes := make(map[string]bool)
			for _, n := range I.Nodes() {
				nn := node(n)
				nn.Attrs["color"] = "red"
				delNodes[nn.DOTID()] = true
			}
			// Dump pre-merge.
			G.SetDOTID(G.DOTID() + "_a")
			createGraph(G)

			// The second order graph, G^2, is derived from G^1 by collapsing each
			// interval in G^1 into a node.

			G = cfg.Merge(G, delNodes, newName)
			name := fmt.Sprintf("G%d_%d", i-1, intNum)
			G.SetDOTID(name)
			createGraph(G)
			intNum++
		}
		name := fmt.Sprintf("G%d", i)
		G.SetDOTID(name)
		createGraph(G)
		Gs = append(Gs, G)
	}
	return Gs
}

// structLoop marks all nodes of G belonging to loops.
func structLoop(G *cfg.Graph) {
	Gs := DerivedGraphSeq(G)
	for _, Gi := range Gs {
		Is := flow.Intervals(Gi, Gi.Entry())
		for _, Ii := range Is {
			latch, ok := findLatch(Ii)
			if !ok {
				continue
			}
			latch.IsLatch = true
			// TODO: Check latching node is at the same nesting level of case
			// statements (if any).

			// Mark nodes belonging to loop and determine type of loop.
			loop(Ii, latch)

			// TODO: add nodes part of loop to inLoop. (latch, Ii.head)
		}
	}
}

// struct2Way marks all nodes of G belonging to 2-way conditionals.
//
// Pre: G is a graph numbered in reverse postorder.
//
// Post: 2-way conditionals are marked in G. the follow node for all 2-way
// conditionals is determined.
func struct2Way(G *cfg.Graph) {
	domtree := path.Dominators(G.Entry(), G)
	// unresolved = {}
	unresolved := make(map[graph.Node]bool)
	// for (all nodes m in N in descending order)
	for _, m := range cfg.SortByRevPost(G.Nodes()) {
		/* TODO: add to the if-statement below && m.LoopHead != m && !m.IsLatch */
		if len(G.From(m)) != 2 {
			continue
		}
		if n, ok := find2WayFollow(G, m, domtree); ok {
			// follow(m) = n
			mm := node(m)
			mm.Follow = n
			// for (all x in unresolved)
			for x := range unresolved {
				// follow(x) = n
				xx := node(x)
				xx.Follow = n
				// unresolved = unresolved - {x}
				delete(unresolved, x)
			}
		} else {
			// unresolved = unresolved U {m}
			unresolved[m] = true
		}
	}
	pretty.Println("unresolved:", unresolved)
}

// find2WayFollow locates the follow node of the 2-way conditional.
func find2WayFollow(G *cfg.Graph, m graph.Node, domtree path.DominatorTree) (graph.Node, bool) {
	// n = max{i | immedDom(i) == m and #inEdges(i) >= 2}
	var n *cfg.Node
	for _, i := range G.Nodes() {
		if domtree.DominatorOf(i) == m && len(G.To(i)) >= 2 {
			ii := node(i)
			if n == nil || n.Post < ii.Post {
				n = ii
			}
		}
	}
	return n, n != nil
}

// loop marks the nodes belonging to the loop determined by (head, latch), and
// determines the loop type.
func loop(I *flow.Interval, latch graph.Node) {
	h := node(I.Head)
	h.InLoop = true
	h.LoopHead = h
	l := node(latch)
	l.InLoop = true
	l.LoopHead = h
	nodes := make(map[graph.Node]bool)
	nodes[h] = true
	// TODO: Consider moving idom computation Structure, and perform on G rather
	// than I.
	domtree := path.Dominators(h, I)
	// Mark nodes in loop headed by head.
	for _, n := range cfg.SortByRevPost(I.Nodes()) {
		nn := node(n)
		if nn.Post <= h.Post || nn.Post >= l.Post {
			continue
		}
		if idom := domtree.DominatorOf(n); !nodes[idom] {
			continue
		}
		nodes[nn] = true
		// Set loop header if not yet part of another loop.
		if nn.LoopHead == nil {
			nn.LoopHead = h
		}
	}
	nodes[l] = true
}

// findLatch returns the latching node of I(h), the node with the greatest
// enclosing back edge to h (if any).
func findLatch(I *flow.Interval) (*cfg.Node, bool) {
	var latch *cfg.Node
	for _, pred := range I.To(I.Head) {
		p := node(pred)
		if I.Has(p) && !p.InLoop {
			if latch == nil {
				latch = p
			} else if p.Post > latch.Post {
				latch = p
			}
		}
	}
	return latch, latch != nil
}

const dir = "_dump_"

func init() {
	if err := os.RemoveAll(dir); err != nil {
		log.Fatalf("%+v", errors.WithStack(err))
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("%+v", errors.WithStack(err))
	}
}

func createGraph(g *cfg.Graph) {
	name := g.DOTID()
	if len(name) == 0 {
		panic(fmt.Errorf("missing name in graph %v", g))
	}
	buf, err := dot.Marshal(g, name, "", "\t", false)
	if err != nil {
		log.Fatalf("%+v", errors.WithStack(err))
	}
	path := filepath.Join(dir, name+".dot")
	if err := ioutil.WriteFile(path, buf, 0644); err != nil {
		log.Fatalf("%+v", errors.WithStack(err))
	}
}

// node asserts that the given node is a control flow graph node.
func node(n graph.Node) *cfg.Node {
	if n, ok := n.(*cfg.Node); ok {
		return n
	}
	panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
}
