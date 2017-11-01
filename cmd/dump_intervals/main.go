package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"log"
	"strconv"

	"gonum.org/v1/gonum/graph"

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
	cfa.CompoundCond(g)
	//spew.Dump(g.Nodes())
	f := genFunc(g)
	//pretty.Println("f:", f)
	buf := &bytes.Buffer{}
	if err := printer.Fprint(buf, token.NewFileSet(), f); err != nil {
		return errors.WithStack(err)
	}
	fmt.Println(buf.String())

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

type generator struct {
	g    *cfg.Graph
	done map[graph.Node]bool
	cur  *ast.BlockStmt
}

func genFunc(g *cfg.Graph) *ast.FuncDecl {
	name := fmt.Sprintf("f_%s", unquote(g.DOTID()))
	gen := &generator{
		g:    g,
		done: make(map[graph.Node]bool),
		cur:  &ast.BlockStmt{},
	}
	entry := node(g.Entry())
	fmt.Println("entry:", entry)
	fmt.Println("entry.Follow:", entry.Follow)
	gen.genCode(entry, entry.Follow)
	return &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		Body: gen.cur,
	}
}

func (gen *generator) genCode(n, ifFollow *cfg.Node) {
	fmt.Println("==> n:", n)
	fmt.Println("==> ifFollow:", ifFollow)
	// Break early if node is the follow node of an if-statement.
	if ifFollow != nil && n == ifFollow {
		return
	}

	// Check if code already generated for block.
	label := ast.NewIdent(fmt.Sprintf("l_%s", unquote(n.DOTID())))
	if gen.done[n] {
		stmt := &ast.BranchStmt{
			Tok:   token.GOTO,
			Label: label,
		}
		gen.cur.List = append(gen.cur.List, stmt)
		return
	}
	gen.done[n] = true

	// TODO: Add support for loops.

	g := gen.g
	succs := g.From(n)
	switch len(succs) {
	// Return statement.
	case 0:
		stmt := &ast.LabeledStmt{
			Label: label,
			Stmt:  &ast.ReturnStmt{},
		}
		gen.cur.List = append(gen.cur.List, stmt)
		return
	// Sequence.
	case 1:
		stmt := &ast.LabeledStmt{
			Label: label,
			Stmt:  &ast.EmptyStmt{},
		}
		gen.cur.List = append(gen.cur.List, stmt)
		gen.genCode(node(succs[0]), ifFollow)
		return
	// Two-way conditional or loop.
	case 2:
		if n.Follow == nil {
			panic(fmt.Errorf("support for unresolved 2-way nodes not yet supported; no follow node for %q", n.DOTID()))
		}
		bak := gen.cur
		t := g.TrueTarget(n)
		f := g.FalseTarget(n)
		switch {
		case t == n.Follow && f == n.Follow:
			panic("support for multiple edges to follow node not yet supported")
		case t == n.Follow:
			// if-then
			//    false branch is body.
			fmt.Println("if:", n.DOTID())
			fmt.Println("   then:", node(f).DOTID())
			body := &ast.BlockStmt{}
			gen.cur = body
			gen.genCode(f, n.Follow)
			stmt := &ast.LabeledStmt{
				Label: label,
				Stmt: &ast.IfStmt{
					Cond: ast.NewIdent("cond"),
					Body: body,
				},
			}
			gen.cur = bak
			gen.cur.List = append(gen.cur.List, stmt)
		case f == n.Follow:
			// if-then
			//    true branch is body.
			fmt.Println("if:", n.DOTID())
			fmt.Println("   then:", node(t).DOTID())
			body := &ast.BlockStmt{}
			gen.cur = body
			gen.genCode(t, n.Follow)
			stmt := &ast.LabeledStmt{
				Label: label,
				Stmt: &ast.IfStmt{
					Cond: ast.NewIdent("cond"),
					Body: body,
				},
			}
			gen.cur = bak
			gen.cur.List = append(gen.cur.List, stmt)
		default:
			// if-else
			fmt.Println("if:", n.DOTID())
			fmt.Println("   then:", node(t).DOTID())
			fmt.Println("   else:", node(f).DOTID())
			trueBody := &ast.BlockStmt{}
			gen.cur = trueBody
			gen.genCode(t, n.Follow)
			falseBody := &ast.BlockStmt{}
			gen.cur = falseBody
			gen.genCode(f, n.Follow)
			stmt := &ast.LabeledStmt{
				Label: label,
				Stmt: &ast.IfStmt{
					Cond: ast.NewIdent("cond"),
					Body: trueBody,
					Else: falseBody,
				},
			}
			gen.cur = bak
			gen.cur.List = append(gen.cur.List, stmt)
		}
		// Continue with the follow.
		fmt.Println("### >> n.Follow", n.Follow)
		gen.genCode(n.Follow, n.Follow.Follow)
	default:
		panic(fmt.Errorf("support for node with %d successors not yet implemented", len(g.From(n))))
	}
}

// ### [ Helper functions ] ####################################################

// node asserts that the given node is a control flow graph node.
func node(n graph.Node) *cfg.Node {
	if n, ok := n.(*cfg.Node); ok {
		return n
	}
	panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
}

// unquote returns an unquoted version of s.
func unquote(s string) string {
	s, err := strconv.Unquote(s)
	if err != nil {
		panic(fmt.Errorf("unable to unquote %q; %v", s, err))
	}
	return s
}
