package cfg

import (
	"fmt"

	"gonum.org/v1/gonum/graph/encoding/dot"
)

// String returns the string representation of the graph in Graphviz DOT format.
func (g *Graph) String() string {
	data, err := dot.Marshal(g, g.DOTID(), "", "\t", false)
	if err != nil {
		panic(fmt.Errorf("unable to marshal control flow graph in DOT format; %v", err))
	}
	return string(data)
}
