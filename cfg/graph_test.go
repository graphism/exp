package cfg

import (
	"io/ioutil"
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
		path  string
		nodes map[string]bool
		id    string
	}{
		{
			path:  "testdata/b.dot",
			nodes: map[string]bool{"B13": true, "B14": true, "B15": true},
			id:    "I3",
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
		buf, err := ioutil.ReadFile(gold.path + ".golden")
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		want := string(buf)
		// Merge.
		out := Merge(in, gold.nodes, gold.id)
		got := out.String()
		if got != want {
			t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
			continue
		}
	}
}
