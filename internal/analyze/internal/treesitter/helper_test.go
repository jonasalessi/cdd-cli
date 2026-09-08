package treesitter

import (
	"testing"

	"github.com/stretchr/testify/require"
	ts "github.com/tree-sitter/go-tree-sitter"
	tsbind "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// parse parses src with the TypeScript grammar, which is only a convenient
// grammar to exercise grammar-agnostic helpers against; nothing here depends
// on its node names beyond the tests that spell them out.
func parse(t *testing.T, src string) *ts.Tree {
	t.Helper()
	p := ts.NewParser()
	t.Cleanup(p.Close)
	require.NoError(t, p.SetLanguage(ts.NewLanguage(tsbind.LanguageTypescript())))
	tree := p.Parse([]byte(src), nil)
	require.NotNil(t, tree)
	t.Cleanup(tree.Close)
	return tree
}
