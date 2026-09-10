package java

import (
	"context"
	"fmt"
	"path"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/analyze/internal/treesitter"
)

// analyzer counts the Java ICP constructs of one file at a time. It owns one
// tree-sitter parser, which is not safe for concurrent use, so the pipeline
// builds one analyzer per worker and closes it when the worker exits.
type analyzer struct {
	prefixes []string
	grammar  *grammar
	parser   *ts.Parser
}

// NewAnalyzer returns a Java analyzer. The returned value holds native
// resources and implements io.Closer; the pipeline must close it.
func NewAnalyzer(opts analyze.Options) analyze.Analyzer {
	return &analyzer{
		prefixes: opts.InternalPrefixes,
		grammar:  sharedGrammar(),
		parser:   ts.NewParser(),
	}
}

// Close releases the parser. The tree-sitter binding installs no finalizers,
// so anything not closed leaks on the C heap. Close is idempotent.
func (a *analyzer) Close() error {
	if a.parser != nil {
		a.parser.Close()
		a.parser = nil
	}
	return nil
}

// Analyze parses src and returns the raw counts of every unit it contains. A
// file that does not parse yields no units and one warning naming the
// position of the first syntax error (FR-5). Only `.java` is accepted: any
// other extension is an error, never a silent guess.
func (a *analyzer) Analyze(ctx context.Context, p string, src []byte) (analyze.FileResult, error) {
	if ext := path.Ext(p); !strings.EqualFold(ext, extJava) {
		return analyze.FileResult{}, fmt.Errorf("%s: unsupported Java extension %q", p, ext)
	}
	tree, err := a.parse(ctx, src)
	if err != nil {
		return analyze.FileResult{}, fmt.Errorf("%s: %w", p, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root.HasError() {
		return analyze.FileResult{Warnings: []string{treesitter.SyntaxWarning(root)}}, nil
	}
	return analyze.FileResult{Units: a.measure(root, src)}, nil
}

// parse binds the grammar on first use and runs the parser over src within
// the shared parse budget.
func (a *analyzer) parse(ctx context.Context, src []byte) (*ts.Tree, error) {
	if a.parser == nil {
		return nil, fmt.Errorf("analyzer is closed")
	}
	if a.parser.Language() == nil {
		if err := a.parser.SetLanguage(a.grammar.lang); err != nil {
			return nil, fmt.Errorf("set grammar: %w", err)
		}
	}
	return treesitter.Parse(ctx, a.parser, src)
}

// measure counts every unit of a parsed file. Unit extraction and the
// counters land with the rules that need them, so a file that parses reports
// nothing yet.
func (a *analyzer) measure(_ *ts.Node, _ []byte) []analyze.Unit {
	return nil
}
