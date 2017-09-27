package cfg

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	golden := []struct {
		path string
	}{
		{path: "testdata/a.dot"},
	}
	for _, gold := range golden {
		buf, err := ioutil.ReadFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to read file; %v", gold.path, err)
			continue
		}
		want := strings.TrimSpace(string(buf))
		g, err := ParseString(want)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		got := g.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}

func TestCopy(t *testing.T) {
	golden := []struct {
		path string
	}{
		{path: "testdata/a.dot"},
	}
	for _, gold := range golden {
		buf, err := ioutil.ReadFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to read file; %v", gold.path, err)
			continue
		}
		want := strings.TrimSpace(string(buf))
		src, err := ParseString(want)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		dst := NewGraph()
		Copy(dst, src)
		got := dst.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}

func TestMerge(t *testing.T) {
	golden := []struct {
		path     string
		wantPath string
		nodes    map[string]bool
		id       string
	}{
		{
			path:     "testdata/sample.dot",
			wantPath: "testdata/sample.dot.I1.golden",
			nodes:    map[string]bool{"B1": true, "B2": true, "B3": true, "B4": true, "B5": true},
			id:       "I1",
		},
		{
			path:     "testdata/sample.dot",
			wantPath: "testdata/sample.dot.I3.golden",
			nodes:    map[string]bool{"B13": true, "B14": true, "B15": true},
			id:       "I3",
		},
	}
	for _, gold := range golden {
		// Parse input.
		in, err := ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		// Parse golden output.
		buf, err := ioutil.ReadFile(gold.wantPath)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		want := strings.TrimSpace(string(buf))
		// Merge.
		out := Merge(in, gold.nodes, gold.id)
		got := out.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}

func TestInitDFSOrder(t *testing.T) {
	golden := []struct {
		path string
		want map[string]int
	}{
		{
			// Sample and reverse post-ordering taken from Fig. 2 in C. Cifuentes'
			// Structuring decompiled graphs [1].
			//
			// [1]: https://pdfs.semanticscholar.org/48bf/d31773af7b67f9d1b003b8b8ac889f08271f.pdf
			path: "testdata/sample.dot",
			want: map[string]int{
				"B1":  1,
				"B2":  2,
				"B3":  4,
				"B4":  3,
				"B5":  5,
				"B6":  6,
				"B7":  11,
				"B8":  12,
				"B9":  13,
				"B10": 14,
				"B11": 15,
				"B12": 7,
				"B13": 8,
				"B14": 9,
				"B15": 10,
			},
		},
	}
	for _, gold := range golden {
		// Parse input.
		in, err := ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		// Init pre- and post depth first search order.
		InitDFSOrder(in)
		// Check results.
		got := make(map[string]int)
		for _, n := range in.Nodes() {
			nn, ok := n.(*Node)
			if !ok {
				panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
			}
			// Compute reverse post-ordering.
			got[nn.name] = len(in.Nodes()) - nn.Post
		}
		if !reflect.DeepEqual(got, gold.want) {
			t.Errorf("%q; output mismatch; expected `%v`, got `%v`", gold.path, gold.want, got)
			continue
		}
	}
}

func TestSortByRevPost(t *testing.T) {
	golden := []struct {
		path string
		want []string
	}{
		{
			// Sample and reverse post-ordering taken from Fig. 2 in C. Cifuentes'
			// Structuring decompiled graphs [1].
			//
			// [1]: https://pdfs.semanticscholar.org/48bf/d31773af7b67f9d1b003b8b8ac889f08271f.pdf
			path: "testdata/sample.dot",
			want: []string{"B1", "B2", "B4", "B3", "B5", "B6", "B12", "B13", "B14", "B15", "B7", "B8", "B9", "B10", "B11"},
		},
	}
	for _, gold := range golden {
		// Parse input.
		in, err := ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		// Init pre- and post depth first search order.
		InitDFSOrder(in)
		// Check results.
		var got []string
		for _, n := range SortByRevPost(in.Nodes()) {
			nn, ok := n.(*Node)
			if !ok {
				panic(fmt.Errorf("invalid node type; expected *cfg.Node, got %T", n))
			}
			got = append(got, nn.name)
		}
		if !reflect.DeepEqual(got, gold.want) {
			t.Errorf("%q; output mismatch; expected `%v`, got `%v`", gold.path, gold.want, got)
			continue
		}
	}
}
