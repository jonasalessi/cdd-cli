package treesitter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstErrorNode(t *testing.T) {
	tree := parse(t, "class A {}\nclass {\n")
	root := tree.RootNode()
	require.True(t, root.HasError())
	n := FirstErrorNode(root)
	require.NotNil(t, n)
	line, _ := Position(n)
	require.Equal(t, 2, line)
	require.Equal(t, SyntaxError+" at 2:1", SyntaxWarning(root))
}

func TestFirstErrorNodeOfACleanTree(t *testing.T) {
	tree := parse(t, "class A {}\n")
	root := tree.RootNode()
	require.Nil(t, FirstErrorNode(root))
	require.Equal(t, SyntaxError+" at 1:1", SyntaxWarning(root),
		"a root flagged with no error node falls back to the start of the file")
}
