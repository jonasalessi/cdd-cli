package typescript

import (
	"context"
	"fmt"
	"strconv"
	"time"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
)

// parseBudget is how long a single file may take to parse when the caller
// set no deadline. It bounds the damage a minified or generated file can do
// without turning a slow machine into a failure.
const parseBudget = 30 * time.Second

// syntaxError is the warning a file that does not parse produces.
const syntaxError = "syntax error"

// analyzer counts the TypeScript ICP constructs of one file at a time. It
// owns one tree-sitter parser and one reusable cursor, neither of which is
// safe for concurrent use, so the pipeline builds one analyzer per worker
// and closes it when the worker exits.
type analyzer struct {
	prefixes []string
	grammars *grammars
	parser   *ts.Parser
	cursor   *ts.TreeCursor
	// current is the grammar the parser is set to, so switching languages
	// between files costs nothing when the extension does not change.
	current *grammar
}

// NewAnalyzer returns a TypeScript analyzer. The returned value holds
// native resources and implements io.Closer; the pipeline must close it.
func NewAnalyzer(opts analyze.Options) analyze.Analyzer {
	return &analyzer{
		prefixes: opts.InternalPrefixes,
		grammars: newGrammars(),
		parser:   ts.NewParser(),
	}
}

// Close releases the parser and cursor. The tree-sitter binding installs no
// finalizers, so anything not closed leaks on the C heap. Close is
// idempotent.
func (a *analyzer) Close() error {
	if a.cursor != nil {
		a.cursor.Close()
		a.cursor = nil
	}
	if a.parser != nil {
		a.parser.Close()
		a.parser = nil
	}
	return nil
}

// Analyze parses src with the grammar the extension of path selects and
// returns the raw counts of every unit it contains. A file that does not
// parse yields no units and one warning (FR-5).
func (a *analyzer) Analyze(ctx context.Context, path string, src []byte) (analyze.FileResult, error) {
	g, err := a.grammars.forPath(path)
	if err != nil {
		return analyze.FileResult{}, err
	}
	tree, err := a.parse(ctx, g, src)
	if err != nil {
		return analyze.FileResult{}, fmt.Errorf("%s: %w", path, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root.HasError() {
		return analyze.FileResult{Warnings: []string{syntaxWarning(path, root)}}, nil
	}
	mods := modules(g, root, src, a.prefixes)
	decls := units(g, root, src)
	out := make([]analyze.Unit, 0, len(decls))
	for i := range decls {
		out = append(out, a.measure(g, &decls[i], mods, src))
	}
	return analyze.FileResult{Units: out}, nil
}

// parse runs the parser over src, bounded by the caller's deadline when
// there is one and by parseBudget when there is not.
func (a *analyzer) parse(ctx context.Context, g *grammar, src []byte) (*ts.Tree, error) {
	if a.parser == nil {
		return nil, fmt.Errorf("analyzer is closed")
	}
	if a.current != g {
		if err := a.parser.SetLanguage(g.lang); err != nil {
			return nil, fmt.Errorf("set grammar: %w", err)
		}
		a.current = g
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// The parser is bounded by its own timeout rather than by the context:
	// the binding's context-aware entry point dereferences a cancellation
	// flag the caller never set. The timeout is derived from the deadline,
	// so a run that is running out of time does not wait for a whole file.
	a.parser.SetTimeoutMicros(timeoutMicros(budget(ctx)))
	tree := a.parser.Parse(src, nil)
	if tree == nil {
		// The parser stopped halfway; reset it so the next file starts
		// from the beginning instead of resuming this one.
		a.parser.Reset()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("parsing did not finish within %s", budget(ctx))
	}
	return tree, nil
}

// budget returns how long parsing may take: what is left of the caller's
// deadline, or parseBudget when the caller set none.
func budget(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return parseBudget
	}
	switch left := time.Until(deadline); {
	case left <= 0:
		return time.Microsecond
	case left < parseBudget:
		return left
	default:
		return parseBudget
	}
}

// timeoutMicros turns a parse budget into the unsigned microseconds the
// parser expects, never below one.
func timeoutMicros(d time.Duration) uint64 {
	if micros := d.Microseconds(); micros > 1 {
		return uint64(micros)
	}
	return 1
}

// measure counts one unit and attributes the file's imports to it.
func (a *analyzer) measure(g *grammar, d *unitDecl, mods []module, src []byte) analyze.Unit {
	c := newCounter(g, src, d)
	walk(a.treeCursor(&d.node), &d.node, c.visit)
	c.countCoupling(mods)
	return analyze.Unit{Name: d.name, Kind: d.kind, Line: d.line, Col: d.col, Counts: c.counts}
}

// treeCursor returns the analyzer's cursor, creating it on first use. A
// cursor is not bound to the tree it was created from: walk resets it onto
// whichever node it is given.
func (a *analyzer) treeCursor(n *ts.Node) *ts.TreeCursor {
	if a.cursor == nil {
		a.cursor = n.Walk()
	}
	return a.cursor
}

// syntaxWarning names the file and the position of the first error or
// missing node, 1-based.
func syntaxWarning(path string, root *ts.Node) string {
	line, col := 1, 1
	if n := firstErrorNode(root); n != nil {
		line, col = position(n)
	}
	return path + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(col) + ": " + syntaxError
}
