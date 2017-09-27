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
	"github.com/pkg/errors"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

func Structure(g *cfg.Graph) []*cfg.Graph {
	Gs := DerivedGraphSeq(g)
	//loopStruct(g, Gs)
	return Gs
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
				nn, ok := n.(dot.Node)
				if !ok {
					panic(fmt.Errorf("invalid node type; expected dot.Node, got %T", n))
				}
				delNodes[nn.DOTID()] = true
			}
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

func loopStruct(G *cfg.Graph, Gs []*cfg.Graph) {
	inLoop := make(map[graph.Node]bool)
	for _, Gi := range Gs {
		Is := flow.Intervals(Gi, Gi.Entry())
		for _, Ii := range Is {
			latch, ok := findLatch(Ii, inLoop)
			if !ok {
				continue
			}
			// Mark nodes belonging to loop.
			loop(Ii, latch, inLoop)
			//inLoop[Ii.Head] = true
			//inLoop[latch] = true

			// TODO: add nodes part of loop to inLoop. (latch, Ii.head)
		}
	}
}

func loop(I *flow.Interval, latch graph.Node, inLoop map[graph.Node]bool) {
	inLoop[I.Head] = true
	inLoop[latch] = true
}

func findLatch(I *flow.Interval, inLoop map[graph.Node]bool) (graph.Node, bool) {
	for _, pred := range I.To(I.Head) {
		if I.Has(pred) && !inLoop[pred] {
			return pred, true
		}
	}
	return nil, false
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
