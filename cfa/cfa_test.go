package cfa

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/graphism/exp/cfg"
)

func TestDerivedGraphSeq(t *testing.T) {
	golden := []struct {
		path string
		want []string
	}{
		{
			path: "testdata/sample.dot",
			want: []string{
				"testdata/sample.dot.G1.golden",
				"testdata/sample.dot.G2.golden",
				"testdata/sample.dot.G3.golden",
				"testdata/sample.dot.G4.golden",
			},
		},
	}
	for _, gold := range golden {
		in, err := cfg.ParseFile(gold.path)
		if err != nil {
			t.Errorf("%q; unable to parse file; %v", gold.path, err)
			continue
		}
		gs := DerivedGraphSeq(in)
		if len(gs) != len(gold.want) {
			t.Errorf("%q: number of derived graphs mismatch; expected %d, got %d", gold.path, len(gold.want), len(gs))
			continue
		}
		for i, g := range gs {
			buf, err := ioutil.ReadFile(gold.want[i])
			if err != nil {
				t.Errorf("%q; unable to read file; %v", gold.path, err)
				continue
			}
			want := strings.TrimSpace(string(buf))
			got := g.String()
			if got != want {
				t.Errorf("%q; output mismatch; expected `%s`, got `%s`", gold.path, want, got)
				continue
			}
		}
	}
}
