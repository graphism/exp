package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/decomp/decomp/graph/cfg"
	"github.com/graphism/exp/flow"
	"github.com/pkg/errors"
)

func main() {
	flag.Parse()
	for _, path := range flag.Args() {
		if err := dumpIntervals(path); err != nil {
			log.Fatalf("%+v", err)
		}
	}
}

func dumpIntervals(path string) error {
	fmt.Printf("\n=== [ %s ] ===\n\n", path)
	g, err := cfg.ParseFile(path)
	if err != nil {
		return errors.WithStack(err)
	}
	is := flow.Intervals(g, g.Entry())
	for _, i := range is {
		fmt.Println("head:", i.Head)
		for _, n := range i.Nodes() {
			fmt.Println("   n:", n)
		}
	}
	return nil
}
