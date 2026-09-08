package treesitter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSpanOf pins the 1-based, end-exclusive convention analyze.Occurrence
// carries.
func TestSpanOf(t *testing.T) {
	tree := parse(t, "\n  class A {}\n")
	class := tree.RootNode().NamedChild(0)
	require.Equal(t, Span{Line: 2, Col: 3, EndLine: 2, EndCol: 13}, SpanOf(class))
	line, col := Position(class)
	require.Equal(t, 2, line)
	require.Equal(t, 3, col)
}
