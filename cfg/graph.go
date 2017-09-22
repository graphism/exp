// Package cfg provides access to control flow graphs.
package cfg

import (
	"fmt"
	"sort"

	"github.com/graphism/simple"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// === [ Graph ] ===============================================================

// Graph is a control flow graph.
type Graph struct {
	*simple.DirectedGraph
	// Graph ID.
	id string
	// Entry node of the control flow graph.
	entry graph.Node
	// nodes maps from node ID to graph node.
	nodes map[string]*Node
}

// NewGraph returns a new control flow graph.
func NewGraph() *Graph {
	return &Graph{
		DirectedGraph: simple.NewDirectedGraph(),
		nodes:         make(map[string]*Node),
	}
}

// String returns the string representation of the graph in Graphviz DOT format.
func (g *Graph) String() string {
	data, err := dot.Marshal(g, g.DOTID(), "", "\t", false)
	if err != nil {
		panic(fmt.Errorf("unable to marshal control flow graph in DOT format; %v", err))
	}
	return string(data)
}

// initNodes initializes the mapping between node IDs and graph nodes.
func (g *Graph) initNodes() {
	for _, n := range g.Nodes() {
		nn, ok := n.(*Node)
		if !ok {
			panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
		}
		if len(nn.id) == 0 {
			panic(fmt.Errorf("invalid node; missing node ID in %#v", nn))
		}
		if prev, ok := g.nodes[nn.id]; ok {
			panic(fmt.Errorf("node ID %q already present in graph; prev node %#v, new node %#v", nn.id, prev, nn))
		}
		g.nodes[nn.id] = nn
	}
}

// --- [ dot.Graph ] -----------------------------------------------------------

// DOTID returns the DOT ID of the graph.
func (g *Graph) DOTID() string {
	return g.id
}

// --- [ dot.DOTIDSetter ] -----------------------------------------------------

// SetDOTID sets the DOT ID of the graph.
func (g *Graph) SetDOTID(id string) {
	g.id = id
}

// --- [ graph.Builder ] -------------------------------------------------------

// ~~~ [ graph.NodeAdder ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewNode returns a new node with a unique arbitrary ID.
func (g *Graph) NewNode() graph.Node {
	return &Node{
		Node:  g.DirectedGraph.NewNode(),
		Attrs: make(Attrs),
	}
}

// AddNode adds a node to the graph.
//
// If the added node ID matches an existing node ID, AddNode will panic.
func (g *Graph) AddNode(n graph.Node) {
	nn, ok := n.(*Node)
	if !ok {
		panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
	}
	g.DirectedGraph.AddNode(nn)
	if nn.entry {
		if g.entry != nil {
			panic(fmt.Errorf("entry node already set in graph; prev entry node %#v, new entry node %#v", g.entry, nn))
		}
		g.entry = nn
	}
}

// ~~~ [ graph.EdgeAdder ] ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewEdge returns a new edge from the source to the destination node.
func (g *Graph) NewEdge(from, to graph.Node) graph.Edge {
	return &Edge{
		Edge:  g.DirectedGraph.NewEdge(from, to),
		Attrs: make(Attrs),
	}
}

// SetEdge adds an edge from one node to another.
//
// If the graph supports node addition the nodes will be added if they do not
// exist, otherwise SetEdge will panic.
func (g *Graph) SetEdge(e graph.Edge) {
	ee, ok := e.(*Edge)
	if !ok {
		panic(fmt.Errorf("invalid edge type; expected *cfg.Edge, got %T", e))
	}
	// Add nodes if not yet present in graph.
	from, to := ee.From(), ee.To()
	if !g.Has(from) {
		g.AddNode(from)
	}
	if !g.Has(to) {
		g.AddNode(to)
	}
	// Add edge.
	g.DirectedGraph.SetEdge(ee)
}

// === [ Node ] ================================================================

// Node is a node in a control flow graph.
type Node struct {
	graph.Node
	// Node ID (e.g. basic block label).
	id string
	// entry specifies whether the node is the entry node of the control flow
	// graph.
	entry bool
	// DOT attributes.
	Attrs
}

// --- [ dot.Node ] ------------------------------------------------------------

// DOTID returns the DOT ID of the node.
func (n *Node) DOTID() string {
	return n.id
}

// --- [ dot.DOTIDSetter ] -----------------------------------------------------

// SetDOTID sets the DOT ID of the node.
func (n *Node) SetDOTID(id string) {
	n.id = id
}

// --- [ encoding.Attributer ] -------------------------------------------------

// Attributes returns the DOT attributes of the node.
func (n *Node) Attributes() []encoding.Attribute {
	if n.entry {
		if prev, ok := n.Attrs["label"]; ok && prev != "entry" {
			panic(fmt.Errorf(`invalid DOT label of entry node; expected "entry", got %q`, prev))
		}
		n.Attrs["label"] = "entry"
	}
	return n.Attrs.Attributes()
}

// --- [ encoding.AttributeSetter ] -------------------------------------------

// SetAttribute sets the DOT attribute of the node.
func (n *Node) SetAttribute(attr encoding.Attribute) error {
	if attr.Key == "label" && attr.Value == "entry" {
		if prev, ok := n.Attrs["label"]; ok && prev != "entry" {
			panic(fmt.Errorf(`invalid DOT label of entry node; expected "entry", got %q`, prev))
		}
		n.entry = true
	} else {
		n.Attrs[attr.Key] = attr.Value
	}
	return nil
}

// === [ Edge ] ================================================================

// Edge is an edge in a control flow graph.
type Edge struct {
	graph.Edge
	// Edge label.
	label string
	// DOT attributes.
	Attrs
}

// --- [ encoding.Attributer ] -------------------------------------------------

// Attributes returns the DOT attributes of the edge.
func (e *Edge) Attributes() []encoding.Attribute {
	if len(e.label) > 0 {
		if prev, ok := e.Attrs["label"]; ok && prev != e.label {
			panic(fmt.Errorf(`mismatch of edge DOT label; expected %q, got %q`, e.label, prev))
		}
		e.Attrs["label"] = e.label
	}
	return e.Attrs.Attributes()
}

// --- [ encoding.AttributeSetter ] -------------------------------------------

// SetAttribute sets the DOT attribute of the edge.
func (e *Edge) SetAttribute(attr encoding.Attribute) error {
	e.Attrs[attr.Key] = attr.Value
	return nil
}

// ### [ Helper functions ] ####################################################

// Attrs specifies a set of DOT attributes as key-value pairs.
type Attrs map[string]string

// --- [ encoding.Attributer ] -------------------------------------------------

// Attributes returns the DOT attributes of a node or edge.
func (a Attrs) Attributes() []encoding.Attribute {
	var keys []string
	for key := range a {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var attrs []encoding.Attribute
	for _, key := range keys {
		attr := encoding.Attribute{
			Key:   key,
			Value: a[key],
		}
		attrs = append(attrs, attr)
	}
	return attrs
}
