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
