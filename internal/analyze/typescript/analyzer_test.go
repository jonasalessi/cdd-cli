package typescript

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
)

// TestGrammarSelection is the parse spike (T1): the extension picks the
// grammar, and the two grammars really are mutually exclusive. The same
// source parses cleanly under one extension and fails under the other.
func TestGrammarSelection(t *testing.T) {
	cases := []struct {
		name      string
		fixture   string
		path      string
		wantUnits int
	}{
		{"jsx under the tsx grammar", "component.tsx", "component.tsx", 1},
		{"jsx under the plain grammar", "component.tsx", "component.ts", 0},
		{"angle-bracket cast under the plain grammar", "cast.ts", "cast.ts", 1},
		{"angle-bracket cast under the tsx grammar", "cast.ts", "cast.tsx", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := newTestAnalyzer(t)
			res, err := a.Analyze(context.Background(), c.path, readFixture(t, c.fixture))
			require.NoError(t, err)
			require.Len(t, res.Units, c.wantUnits)
			if c.wantUnits == 0 {
				require.Len(t, res.Warnings, 1)
				require.Contains(t, res.Warnings[0], c.path+":")
				require.Contains(t, res.Warnings[0], syntaxError)
				return
			}
			require.Empty(t, res.Warnings)
		})
	}
}

// TestSyntaxError pins FR-5: no units, one warning naming the file and the
// first error position.
func TestSyntaxError(t *testing.T) {
	res := analyzeFixture(t, "broken.ts")
	require.Empty(t, res.Units)
	require.Equal(t, []string{"broken.ts:2:3: " + syntaxError}, res.Warnings)
}

// TestExtensions checks the grammar table, including the extensions that
// only exist for module resolution.
func TestExtensions(t *testing.T) {
	gs := newGrammars()
	for _, ext := range []string{extTS, extMTS, extCTS, ".TS"} {
		g, err := gs.forPath("dir/file" + ext)
		require.NoError(t, err)
		require.Same(t, gs.plain, g)
	}
	g, err := gs.forPath("dir/file" + extTSX)
	require.NoError(t, err)
	require.Same(t, gs.tsx, g)
}

// TestUnsupportedExtension refuses to guess a grammar.
func TestUnsupportedExtension(t *testing.T) {
	a := newTestAnalyzer(t)
	_, err := a.Analyze(context.Background(), "src/main.js", []byte("const a = 1;\n"))
	require.ErrorContains(t, err, "src/main.js")
}

// TestKindsResolve makes sure every node kind the analyzer looks for exists
// in both grammars, so a grammar upgrade that renames one fails here rather
// than silently counting nothing.
func TestKindsResolve(t *testing.T) {
	gs := newGrammars()
	for _, g := range []*grammar{gs.plain, gs.tsx} {
		for k := kindOther + 1; k < kindCount; k++ {
			name := kindNames[k]
			require.NotEmpty(t, name, "kind %d has no grammar name", k)
			id := g.lang.IdForNodeKind(name, true)
			require.NotZero(t, id, "node kind %q is unknown to the grammar", name)
			require.Equal(t, k, g.byID[id])
		}
	}
}

// TestFieldsResolve does the same for the field names the analyzer
// navigates by.
func TestFieldsResolve(t *testing.T) {
	names := []string{
		fieldOperator, fieldLeft, fieldRight, fieldArgument, fieldName,
		fieldAlias, fieldValue, fieldDeclaration, fieldSource, fieldType,
	}
	gs := newGrammars()
	for _, g := range []*grammar{gs.plain, gs.tsx} {
		for _, name := range names {
			require.NotZero(t, g.lang.FieldIdForName(name), "unknown field %q", name)
		}
	}
}

// TestKindOfOutOfRange guards the symbol-id lookup against a node from
// another grammar.
func TestKindOfOutOfRange(t *testing.T) {
	g := newGrammars().plain
	g.byID = g.byID[:1]
	tree := parseWith(t, g, []byte("class A {}\n"))
	defer tree.Close()
	require.Equal(t, kindOther, g.kindOf(tree.RootNode()))
}

// parseWith parses src with one grammar, for the tests that need a tree
// without going through the analyzer.
func parseWith(t *testing.T, g *grammar, src []byte) *ts.Tree {
	t.Helper()
	p := ts.NewParser()
	t.Cleanup(p.Close)
	require.NoError(t, p.SetLanguage(g.lang))
	tree := p.Parse(src, nil)
	require.NotNil(t, tree)
	return tree
}

// TestReanalyzeDoesNotLeak analyzes the same file a thousand times through
// one analyzer. The binding installs no finalizers, so a parser, tree or
// cursor that is never closed leaks on the C heap; running the loop under
// -race and watching it stay correct is the cheap guard against that.
func TestReanalyzeDoesNotLeak(t *testing.T) {
	a := newTestAnalyzer(t)
	src := readFixture(t, "units.ts")
	for range 1000 {
		res, err := a.Analyze(context.Background(), "units.ts", src)
		require.NoError(t, err)
		require.Len(t, res.Units, 15)
	}
}

// TestAlternatingGrammars switches grammars between files, which resets the
// parser, and checks both keep working.
func TestAlternatingGrammars(t *testing.T) {
	a := newTestAnalyzer(t)
	plain := readFixture(t, "cast.ts")
	tsx := readFixture(t, "component.tsx")
	for range 10 {
		res, err := a.Analyze(context.Background(), "cast.ts", plain)
		require.NoError(t, err)
		require.Len(t, res.Units, 1)
		res, err = a.Analyze(context.Background(), "component.tsx", tsx)
		require.NoError(t, err)
		require.Len(t, res.Units, 1)
	}
}

// TestCloseIsIdempotent covers the pipeline closing an analyzer twice, and
// using one after it was closed.
func TestCloseIsIdempotent(t *testing.T) {
	a := NewAnalyzer(analyze.Options{})
	closer, ok := a.(io.Closer)
	require.True(t, ok)
	require.NoError(t, closer.Close())
	require.NoError(t, closer.Close())
	_, err := a.Analyze(context.Background(), "a.ts", []byte("class A {}\n"))
	require.Error(t, err)
}

// TestCanceledContext stops the parser instead of running to completion.
func TestCanceledContext(t *testing.T) {
	a := newTestAnalyzer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := a.Analyze(ctx, "units.ts", readFixture(t, "units.ts"))
	require.Error(t, err)
}

// TestBudget derives the parse timeout from the caller's deadline.
func TestBudget(t *testing.T) {
	require.Equal(t, parseBudget, budget(context.Background()))

	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	left := budget(short)
	require.Positive(t, left)
	require.Less(t, left, parseBudget)

	past, cancelPast := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelPast()
	require.Equal(t, time.Microsecond, budget(past))

	long, cancelLong := context.WithTimeout(context.Background(), time.Hour)
	defer cancelLong()
	require.Equal(t, parseBudget, budget(long))
}

// TestTimeoutMicros never hands the parser a zero or negative timeout,
// which would mean "no limit at all".
func TestTimeoutMicros(t *testing.T) {
	require.Equal(t, uint64(1), timeoutMicros(0))
	require.Equal(t, uint64(1), timeoutMicros(-time.Second))
	require.Equal(t, uint64(parseBudget.Microseconds()), timeoutMicros(parseBudget))
}
