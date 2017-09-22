package cfg

import "gonum.org/v1/gonum/graph"

// Merge returns a new control flow graph where the specified nodes have been
// collapsed into a single node with the given node ID, and the predecessors and
// successors of the specified nodes.
func Merge(src *Graph, delNodes map[string]bool, newName string) *Graph {
	dst := NewGraph()
	Copy(dst, src)
	preds := make(map[graph.Node]bool)
	succs := make(map[graph.Node]bool)
	newNode := dst.NewNodeWithName(newName)
	dst.AddNode(newNode)
	for delName := range delNodes {
		delNode := dst.nodeWithName(delName)
		// Record predecessors not part of nodes.
		for _, pred := range dst.To(delNode) {
			p := node(pred)
			if !delNodes[p.name] {
				preds[dst.nodeWithName(p.name)] = true
			}
		}
		// Record successors not part of nodes.
		for _, succ := range dst.From(delNode) {
			s := node(succ)
			if !delNodes[s.name] {
				succs[dst.nodeWithName(s.name)] = true
			}
		}
		dst.RemoveNode(delNode)
	}
	// Add edges from predecessors to new node.
	for pred := range preds {
		e := dst.NewEdge(pred, newNode)
		dst.SetEdge(e)
	}
	// Add edges from new node to successors.
	for succ := range succs {
		e := dst.NewEdge(newNode, succ)
		dst.SetEdge(e)
	}
	return dst
}
