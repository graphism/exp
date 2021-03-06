package cfg

// Copy copies nodes and edges as directed edges from the source to the
// destination without first clearing the destination. Copy will panic if a node
// ID in the source graph matches a node ID in the destination.
func Copy(dst, src *Graph) {
	dst.id = src.id
	nodes := src.Nodes()
	for nodes.Next() {
		n := nodes.Node()
		dst.AddNode(n)
	}
	for nodes.Reset(); nodes.Next(); {
		u := nodes.Node()
		vnodes := src.From(u.ID())
		for vnodes.Next() {
			v := vnodes.Node()
			dst.SetEdge(src.Edge(u.ID(), v.ID()))
		}
	}
	dst.initNodes()
}
