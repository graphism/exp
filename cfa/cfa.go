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
	"strconv"
	"strings"

	"github.com/graphism/exp/cfg"
	"github.com/graphism/exp/flow"
	"github.com/mewkiz/pkg/term"
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
	gonumflow "gonum.org/v1/gonum/graph/flow"
)

// dbg logs debug messages to standard error, with the prefix "interval:".
var dbg = log.New(os.Stderr, term.RedBold("interval:")+" ", 0)

func Structure(g *cfg.Graph) {
	cfg.InitDFSOrder(g)
	//structLoops(g)
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
	for i := 2; G.Nodes().Len() > 1; i++ {
		Is := flow.Intervals(G, G.Entry())
		for _, I := range Is {
			// Collapse interval into a single node.
			newName := fmt.Sprintf("I%d", intNum)
			delNodes := make(map[string]bool)
			Inodes := I.Nodes()
			for Inodes.Next() {
				n := Inodes.Node()
				nn := node(n)
				nn.Attrs["fillcolor"] = "red"
				nn.Attrs["style"] = "filled"
				delNodes[nn.DOTID()] = true
			}
			// Dump pre-merge.
			// Store graph DOTID before dump.
			nameBak := G.DOTID()
			G.SetDOTID(nameBak + "_a")
			createGraph(G)
			for Inodes.Reset(); Inodes.Next(); {
				n := Inodes.Node()
				nn := node(n)
				delete(nn.Attrs, "fillcolor")
				delete(nn.Attrs, "style")
			}
			G.SetDOTID(nameBak)

			// The second order graph, G^2, is derived from G^1 by collapsing each
			// interval in G^1 into a node.

			G = cfg.Merge(G, delNodes, newName)
			n, ok := G.NodeWithName(newName)
			if !ok {
				panic(fmt.Errorf("unable to locate new node %q after merge", newName))
			}
			n.Attrs["fillcolor"] = "red"
			n.Attrs["style"] = "filled"
			name := fmt.Sprintf("G%d_b_%d", i-1, intNum)
			G.SetDOTID(name)
			createGraph(G)
			delete(n.Attrs, "fillcolor")
			delete(n.Attrs, "style")
			intNum++
		}
		name := fmt.Sprintf("G%d", i)
		G.SetDOTID(name)
		createGraph(G)
		Gs = append(Gs, G)
	}
	return Gs
}

// structLoops marks all nodes of G belonging to loops.
func structLoops(G *cfg.Graph) {
	Gs := DerivedGraphSeq(G)
	for _, Gi := range Gs {
		cfg.InitDFSOrder(Gi)
		Is := flow.Intervals(Gi, Gi.Entry())
		for _, Ii := range Is {
			// Find latch node of loop.
			latch, ok := findLatch(Ii)
			if !ok {
				continue
			}
			dbg.Println("latch:", latch)
			head := node(Ii.Head)
			head.Latch = latch

			// TODO: Check latching node is at the same nesting level of case
			// statements (if any).
			// Mark nodes belonging to loop and determine type of loop.
			loop(Ii, latch)
			latch.IsLatch = true
		}
	}
}

// findLatch returns the latching node of I(h), the node with the greatest
// enclosing back edge to h (if any).
func findLatch(I *flow.Interval) (*cfg.Node, bool) {
	var latch *cfg.Node
	// Find greatest enclosing back edge (if any).
	predNodes := I.To(I.Head.ID())
	for predNodes.Next() {
		pred := predNodes.Node()
		if I.Node(pred.ID()) == nil {
			continue
		}
		p, h := node(pred), node(I.Head)
		if !isBackEdge(p, h) {
			continue
		}
		if latch == nil {
			latch = p
		} else if p.RevPost > latch.RevPost {
			latch = p
		}
	}
	return latch, latch != nil
}

// isBackEdge reports whether (pred, head) is a back edge. If head was visited
// first during depth first search traversal (i.e. has a smaller Pre number), or
// head == pred, then it is a back edge.
func isBackEdge(pred, head *cfg.Node) bool {
	return head.Pre < pred.Pre
}

// loop marks the nodes belonging to the loop determined by (latch, head), and
// determines the loop type.
func loop(I *flow.Interval, latch *cfg.Node) {
	head := node(I.Head)
	head.LoopHead = head
	// nodes belonging to loop.
	nodes := make(map[graph.Node]bool)
	nodes[head] = true
	// TODO: Consider moving idom computation Structure, and perform on G rather
	// than I.
	domtree := gonumflow.Dominators(head, I)
	// Mark nodes in loop headed by head.
	for _, n := range cfg.SortByRevPost(graph.NodesOf(I.Nodes())) {
		nn := node(n)
		if nn.RevPost <= head.RevPost {
			continue
		}
		if nn.RevPost >= latch.RevPost {
			break
		}
		if idom := domtree.DominatorOf(n.ID()); !nodes[idom] {
			continue
		}
		nodes[nn] = true
		// Set loop header if not yet part of another loop.
		if nn.LoopHead == nil {
			nn.LoopHead = head
		}
	}
	latch.LoopHead = head
	nodes[latch] = true

	// Determine loop type.
	switch {
	// 2-way latch node.
	case I.From(latch.ID()).Len() == 2:
		switch {
		// 1-way header node.
		case I.From(head.ID()).Len() == 1:
			head.LoopType = cfg.LoopTypePostTest
		// 2-way header node.
		default:
			// use heuristic to determine best type of loop.
			panic("loop type detection heuristic not yet implemented for 2-way header node, 2-way latch node loops")
		}
	// 1-way latch node.
	default:
		switch {
		// 2-way header node.
		case I.From(head.ID()).Len() == 2:
			head.LoopType = cfg.LoopTypePreTest
		// 1-way header node.
		default:
			dbg.Println("latch:", latch)
			dbg.Println("head:", head)
			head.LoopType = cfg.LoopTypeEndless
		}
	}

	// Determine loop follow.
	switch head.LoopType {
	case cfg.LoopTypePreTest:
		// Follow node is the successor of the header node not part of loop nodes.
		succs := graph.NodesOf(I.From(head.ID()))
		if nodes[succs[0]] {
			head.LoopFollow = node(succs[1])
		} else {
			head.LoopFollow = node(succs[0])
		}
	case cfg.LoopTypePostTest:
		// Follow node is the successor of the latch node not part of loop nodes.
		succs := graph.NodesOf(I.From(latch.ID()))
		if nodes[succs[0]] {
			head.LoopFollow = node(succs[1])
		} else {
			head.LoopFollow = node(succs[0])
		}
	case cfg.LoopTypeEndless:
		// Determine follow node (if any) by traversing all nodes in the loop.
		panic("determination of follow node for endless loops not yet implemented")
	}
}

// struct2Way marks all nodes of G belonging to 2-way conditionals.
//
// Pre: G is a graph numbered in reverse postorder.
//
// Post: 2-way conditionals are marked in G. the follow node for all 2-way
// conditionals is determined.
func struct2Way(G *cfg.Graph) {
	domtree := gonumflow.Dominators(G.Entry(), G)
	// unresolved = {}
	unresolved := make(map[graph.Node]bool)

	// Analyze in descending order (note that descending reverse postorder is
	// equivalent to ascending postorder) since it is desirable to analyze the
	// innermost nested conditional first, and then the outer ones.

	// for (all nodes m in N in descending order)
	for _, m := range cfg.SortByPost(graph.NodesOf(G.Nodes())) {
		mm := node(m)
		//dbg.Println("mm:", mm.RevPost, mm.DOTID())
		if G.From(m.ID()).Len() != 2 {
			continue
		}
		if mm.LoopHead == m {
			continue
		}
		if mm.IsLatch {
			continue
		}
		if n, ok := find2WayFollow(G, m, domtree); ok {
			// follow(m) = n
			mm.IfFollow = n
			// for (all x in unresolved)
			for x := range unresolved {
				// follow(x) = n
				xx := node(x)
				xx.IfFollow = n
				// unresolved = unresolved - {x}
				delete(unresolved, x)
			}
		} else {
			// unresolved nodes may be conditionals nested in another conditional
			// structure.

			// unresolved = unresolved U {m}
			unresolved[m] = true
		}
	}
	//pretty.Println("unresolved:", unresolved)
}

// find2WayFollow locates the follow node of the 2-way conditional.
func find2WayFollow(G *cfg.Graph, m graph.Node, domtree gonumflow.DominatorTree) (*cfg.Node, bool) {
	// n = max{i | immedDom(i) == m and #inEdges(i) >= 2}
	//mm := node(m)
	var n *cfg.Node
	for _, i := range cfg.SortByRevPost(graph.NodesOf(G.Nodes())) {
		if domtree.DominatorOf(i.ID()) == m && G.To(i.ID()).Len() >= 2 {
			ii := node(i)
			//dbg.Printf("immdom of %v is %v\n", ii.DOTID(), mm.DOTID())
			if n == nil || ii.RevPost > n.RevPost {
				n = ii
			}
		}
	}
	return n, n != nil
}

// CompoundCond merges the basic blocks of compound conditions into single basic
// blocks.
func CompoundCond(g *cfg.Graph) *cfg.Graph {
	change := true
	for change {
		change = false
		// Traverse nodes in postorder, this way, the header node of a compound
		// condition is analyzed first.
		for _, n := range cfg.SortByRevPost(graph.NodesOf(g.Nodes())) {
			if g.From(n.ID()).Len() != 2 {
				continue
			}
			nn := node(n)
			switch {
			case compoundCondAND(g, nn):
				dbg.Println("AND located at:", nn)
				x := nn
				y := g.TrueTarget(x)
				e := g.FalseTarget(x)
				t := g.TrueTarget(y)
				g = mergeCond(g, x, y, e, t, "CondAND")
				change = true
			case compoundCondOR(g, nn):
				dbg.Println("OR located at:", nn)
				x := nn
				t := g.TrueTarget(x)
				y := g.FalseTarget(x)
				e := g.FalseTarget(y)
				g = mergeCond(g, x, y, e, t, "CondOR")
				change = true
			case compoundCondNAND(g, nn):
				dbg.Println("NAND located at:", nn)
				x := nn
				e := g.TrueTarget(x)
				y := g.FalseTarget(x)
				t := g.TrueTarget(y)
				g = mergeCond(g, x, y, e, t, "CondNAND")
				change = true
			case compoundCondNOR(g, nn):
				dbg.Println("NOR located at:", nn)
				x := nn
				y := g.TrueTarget(x)
				t := g.FalseTarget(x)
				e := g.FalseTarget(y)
				g = mergeCond(g, x, y, e, t, "CondNOR")
				change = true
			}
		}
	}
	return g
}

// compoundCondAND reports whether a compound AND condition is headed at the
// given node.
func compoundCondAND(g *cfg.Graph, x *cfg.Node) bool {
	// Check (x && y) case. The left and right edge represent the false and true
	// branch, respectively, in the illustration below.
	//
	//    x AND y
	//
	//    x
	//    ↓ ↘
	//    ↓   y
	//    ↓ ↙   ↘
	//    e       t
	//
	y := g.TrueTarget(x)  // true branch
	e := g.FalseTarget(x) // false branch
	if g.To(y.ID()).Len() == 1 && g.From(y.ID()).Len() == 2 {
		t := g.TrueTarget(y)   // true branch
		e2 := g.FalseTarget(y) // false branch
		if e == e2 {
			return true
		}
		_ = t
	}
	return false
}

// compoundCondOR reports whether a compound OR condition is headed at the given
// node.
func compoundCondOR(g *cfg.Graph, x *cfg.Node) bool {
	// Check (x || y) case. The left and right edge represent the false and true
	// branch, respectively, in the illustration below.
	//
	//    x OR y
	//
	//            x
	//          ↙ ↓
	//        y   ↓
	//      ↙   ↘ ↓
	//    e       t
	//
	t := g.TrueTarget(x)  // true branch
	y := g.FalseTarget(x) // false branch
	if g.To(y.ID()).Len() == 1 && g.From(y.ID()).Len() == 2 {
		t2 := g.TrueTarget(y) // true branch
		e := g.FalseTarget(y) // false branch
		if t == t2 {
			return true
		}
		_ = e
	}
	return false
}

// compoundCondNAND reports whether a compound NAND condition is headed at the
// given node.
func compoundCondNAND(g *cfg.Graph, x *cfg.Node) bool {
	// Check (!x && y) case. The left and right edge represent the false and true
	// branch, respectively, in the illustration below.
	//
	//    !x AND y
	//
	//            x
	//          ↙↙
	//        y↙
	//      ↙↙  ↘
	//    e       t
	//
	e := g.TrueTarget(x)  // true branch
	y := g.FalseTarget(x) // false branch
	if g.To(y.ID()).Len() == 1 && g.From(y.ID()).Len() == 2 {
		t := g.TrueTarget(y)   // true branch
		e2 := g.FalseTarget(y) // false branch
		if e == e2 {
			return true
		}
		_ = t
	}
	return false
}

// compoundCondNOR reports whether a compound NOR condition is headed at the
// given node.
func compoundCondNOR(g *cfg.Graph, x *cfg.Node) bool {
	// Check (!x || y) case. The left and right edge represent the false and true
	// branch, respectively, in the illustration below.
	//
	//    !x OR y
	//
	//    x
	//     ↘↘
	//       ↘y
	//      ↙  ↘↘
	//    e       t
	//
	y := g.TrueTarget(x)  // true branch
	t := g.FalseTarget(x) // false branch
	if g.To(y.ID()).Len() == 1 && g.From(y.ID()).Len() == 2 {
		t2 := g.TrueTarget(y) // true branch
		e := g.FalseTarget(y) // false branch
		if t == t2 {
			return true
		}
		_ = e
	}
	return false
}

// mergeCond merges the nodes x and y of the given compound condition.
//
// Example merge for x AND y.
//
// Before
//    x
//    ↓ ↘
//    ↓   y
//    ↓ ↙   ↘
//    e       t
//
// After
//       x&&y
//      ↙    ↘
//    e        t
func mergeCond(g *cfg.Graph, x, y, e, t *cfg.Node, name string) *cfg.Graph {
	// Replace x and y node with new (x AND y) node.
	delNodes := map[string]bool{
		x.DOTID(): true,
		y.DOTID(): true,
	}
	newName := fmt.Sprintf("%s_%s", unquote(x.DOTID()), name)
	g = cfg.Merge(g, delNodes, newName)
	n, ok := g.NodeWithName(newName)
	if !ok {
		panic(fmt.Errorf("unable to locate compound condition node %q", newName))
	}
	trueEdge := edge(g.Edge(n.ID(), t.ID()))
	falseEdge := edge(g.Edge(n.ID(), e.ID()))
	trueEdge.Attrs["label"] = "true"
	falseEdge.Attrs["label"] = "false"
	return g
}

// ### [ Helper functions ] ####################################################

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
	buf, err := dot.Marshal(g, name, "", "\t")
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

// edge asserts that the given edge is a control flow graph edge.
func edge(e graph.Edge) *cfg.Edge {
	if e, ok := e.(*cfg.Edge); ok {
		return e
	}
	panic(fmt.Errorf("invalid edge type; expected *cfg.Edge, got %T", e))
}

// unquote returns an unquoted version of s.
func unquote(s string) string {
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s, err := strconv.Unquote(s)
		if err != nil {
			panic(fmt.Errorf("unable to unquote %q; %v", s, err))
		}
		return s
	}
	return s
}
