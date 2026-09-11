package golang

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
)

// newTestAnalyzer returns an analyzer built the way the pipeline builds one.
// Unlike the tree-sitter analyzers it holds no native resource, so there is
// nothing to close when the test ends.
func newTestAnalyzer(t *testing.T, prefixes ...string) analyze.Analyzer {
	t.Helper()
	return NewAnalyzer(analyze.Options{InternalPrefixes: prefixes})
}

// readFixture returns the content of a file under testdata.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return src
}

// analyzeFixture analyzes a file under testdata under its own name.
func analyzeFixture(t *testing.T, name string, prefixes ...string) analyze.FileResult {
	t.Helper()
	a := newTestAnalyzer(t, prefixes...)
	res, err := a.Analyze(context.Background(), name, readFixture(t, name))
	require.NoError(t, err)
	return res
}

// analyzeSource analyzes an inline source as a .go file that must parse.
func analyzeSource(t *testing.T, src string, prefixes ...string) analyze.FileResult {
	t.Helper()
	a := newTestAnalyzer(t, prefixes...)
	res, err := a.Analyze(context.Background(), "inline"+extGo, []byte(src))
	require.NoError(t, err)
	require.Empty(t, res.Warnings, "inline source must parse:\n%s", src)
	return res
}

// unitNames lists the names of the units of a result, for failure messages.
func unitNames(res analyze.FileResult) []string {
	out := make([]string, 0, len(res.Units))
	for _, u := range res.Units {
		out = append(out, u.Name)
	}
	return out
}
