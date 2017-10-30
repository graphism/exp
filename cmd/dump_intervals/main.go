package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/graphism/exp/cfa"
	"github.com/graphism/exp/cfg"
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
	cfa.Structure(g)
	spew.Dump(g.Nodes())
	//gs := cfa.DerivedGraphSeq(g)
	//for num, g := range gs {
	//	name := fmt.Sprintf("G%d", num)
	//	buf, err := dot.Marshal(g, name, "", "\t", false)
	//	if err != nil {
	//		return errors.WithStack(err)
	//	}
	//	fmt.Println(string(buf))
	//}
	return nil
}
