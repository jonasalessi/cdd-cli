package treesitter

import (
	ts "github.com/tree-sitter/go-tree-sitter"
)

// Walk visits root and every one of its descendants in document order,
// reusing cursor. visit reports whether the walk should descend into the
// node it was given. The walk never leaves root's subtree: the depth
// counter, not the cursor, decides when to stop.
//
// Nodes handed to visit are only valid while the tree that produced them is
// open; copy out anything that has to outlive the walk.
func Walk(cursor *ts.TreeCursor, root *ts.Node, visit func(n *ts.Node) bool) {
	cursor.Reset(*root)
	depth := 0
	for {
		if visit(cursor.Node()) && cursor.GotoFirstChild() {
			depth++
			continue
		}
		for {
			if depth == 0 {
				return
			}
			if cursor.GotoNextSibling() {
				break
			}
			cursor.GotoParent()
			depth--
		}
	}
}

// NamedChildren returns n's named children. It is meant for the small,
// bounded child lists an analyzer inspects by hand (heritage clauses,
// import clauses, top-level declarations), never for a deep traversal.
func NamedChildren(n *ts.Node) []ts.Node {
	count := n.NamedChildCount()
	out := make([]ts.Node, 0, count)
	for i := range count {
		if c := n.NamedChild(i); c != nil {
			out = append(out, *c)
		}
	}
	return out
}

// NamedChildrenInField returns n's named children that sit in the given
// field, which separates `extends Base<T>`'s one parent from its type
// arguments. An empty field takes every named child.
func NamedChildrenInField(n *ts.Node, field string) []ts.Node {
	count := n.NamedChildCount()
	out := make([]ts.Node, 0, count)
	for i := range count {
		c := n.NamedChild(i)
		if c == nil || (field != "" && n.FieldNameForNamedChild(uint32(i)) != field) {
			continue
		}
		out = append(out, *c)
	}
	return out
}

// FirstNamedChild returns n's first named child, nil when it has none.
func FirstNamedChild(n *ts.Node) *ts.Node {
	if n.NamedChildCount() == 0 {
		return nil
	}
	return n.NamedChild(0)
}

// Text returns the source text of n, empty when n is nil.
func Text(n *ts.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(src)
}
