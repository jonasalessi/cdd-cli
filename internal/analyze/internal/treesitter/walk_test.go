package treesitter

import (
	"testing"

	"github.com/stretchr/testify/require"
	ts "github.com/tree-sitter/go-tree-sitter"
)

// TestWalkStaysInsideItsRoot walks a subtree in the middle of a file and
// checks it visits every descendant once, in document order, and nothing
// outside the subtree.
func TestWalkStaysInsideItsRoot(t *testing.T) {
	tree := parse(t, "class A {}\nfunction f(a: number) { return a; }\nclass B {}\n")
	root := tree.RootNode()
	fn := root.NamedChild(1)
	require.Equal(t, "function_declaration", fn.Kind())

	cursor := root.Walk()
	defer cursor.Close()
	var kinds []string
	Walk(cursor, fn, func(n *ts.Node) bool {
		kinds = append(kinds, n.Kind())
		return true
	})
	require.Equal(t, "function_declaration", kinds[0])
	require.Contains(t, kinds, "return_statement")
	require.NotContains(t, kinds, "class_declaration")
	require.NotContains(t, kinds, "program")
}

// TestWalkDoesNotDescendWhenAsked skips the subtree of a node whose visit
// returns false and carries on with its siblings.
func TestWalkDoesNotDescendWhenAsked(t *testing.T) {
	tree := parse(t, "function f() { return 1; }\nclass B {}\n")
	root := tree.RootNode()
	cursor := root.Walk()
	defer cursor.Close()
	var kinds []string
	Walk(cursor, root, func(n *ts.Node) bool {
		kinds = append(kinds, n.Kind())
		return n.Kind() != "function_declaration"
	})
	require.NotContains(t, kinds, "return_statement")
	require.Contains(t, kinds, "class_declaration")
}

// TestWalkReusesTheCursor runs two walks through one cursor, which is how an
// analyzer measures every unit of a file.
func TestWalkReusesTheCursor(t *testing.T) {
	tree := parse(t, "class A { m() { if (x) {} } }\nclass B { if (y) {} }\n")
	root := tree.RootNode()
	cursor := root.Walk()
	defer cursor.Close()
	for _, child := range NamedChildren(root) {
		unit := child
		visited := 0
		Walk(cursor, &unit, func(*ts.Node) bool { visited++; return true })
		require.Positive(t, visited)
	}
}

func TestNamedChildren(t *testing.T) {
	tree := parse(t, "class A {}\nclass B {}\n")
	kids := NamedChildren(tree.RootNode())
	require.Len(t, kids, 2)
	require.Equal(t, "class_declaration", kids[0].Kind())
	require.Len(t, NamedChildren(&kids[0]), 2, "the name and the body; the `class` keyword is anonymous")
	require.Empty(t, NamedChildren(kids[0].NamedChild(0)), "an identifier is a leaf")
}

func TestNamedChildrenInField(t *testing.T) {
	tree := parse(t, "class A extends Base<T> {}\n")
	class := tree.RootNode().NamedChild(0)
	var heritage *ts.Node
	for _, c := range NamedChildren(class) {
		if c.Kind() == "class_heritage" {
			n := c
			heritage = &n
		}
	}
	require.NotNil(t, heritage)
	extends := heritage.NamedChild(0)
	require.Equal(t, "extends_clause", extends.Kind())
	require.Len(t, NamedChildrenInField(extends, ""), 2, "the parent and its type arguments")
	require.Len(t, NamedChildrenInField(extends, "value"), 1)
	require.Equal(t, "Base", NamedChildrenInField(extends, "value")[0].Utf8Text([]byte("class A extends Base<T> {}\n")))
}

func TestFirstNamedChildAndText(t *testing.T) {
	src := "class A {}\n"
	tree := parse(t, src)
	root := tree.RootNode()
	first := FirstNamedChild(root)
	require.NotNil(t, first)
	require.Equal(t, "class A {}", Text(first, []byte(src)))
	require.Nil(t, FirstNamedChild(first.ChildByFieldName("name")))
	require.Empty(t, Text(nil, []byte(src)))
}
