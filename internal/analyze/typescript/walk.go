package typescript

import (
	ts "github.com/tree-sitter/go-tree-sitter"
)

// walk visits root and every one of its descendants in document order,
// reusing cursor. visit reports whether the walk should descend into the
// node it was given. The walk never leaves root's subtree: the depth
// counter, not the cursor, decides when to stop.
//
// Nodes handed to visit are only valid while the tree that produced them is
// open; copy out anything that has to outlive the walk.
func walk(cursor *ts.TreeCursor, root *ts.Node, visit func(n *ts.Node) bool) {
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

// namedChildren returns n's named children. It is meant for the small,
// bounded child lists the analyzer inspects by hand (heritage clauses,
// import clauses, top-level declarations), never for a deep traversal.
func namedChildren(n *ts.Node) []ts.Node {
	count := n.NamedChildCount()
	out := make([]ts.Node, 0, count)
	for i := range count {
		if c := n.NamedChild(i); c != nil {
			out = append(out, *c)
		}
	}
	return out
}

// namedChildrenInField counts n's named children that sit in the given
// field, which separates `extends Base<T>`'s one parent from its type
// arguments.
func namedChildrenInField(n *ts.Node, field string) int {
	count := 0
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if n.FieldNameForNamedChild(uint32(i)) == field {
			count++
		}
	}
	return count
}

// firstNamedChild returns n's first named child, nil when it has none.
func firstNamedChild(n *ts.Node) *ts.Node {
	if n.NamedChildCount() == 0 {
		return nil
	}
	return n.NamedChild(0)
}

// text returns the source text of n, empty when n is nil.
func text(n *ts.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(src)
}

// position returns n's 1-based line and column.
func position(n *ts.Node) (line, col int) {
	p := n.StartPosition()
	return int(p.Row) + 1, int(p.Column) + 1
}

// firstErrorNode returns the first ERROR or MISSING node in n's subtree in
// document order, nil when the subtree is clean.
func firstErrorNode(n *ts.Node) *ts.Node {
	if n.IsError() || n.IsMissing() {
		return n
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || !c.HasError() {
			continue
		}
		if found := firstErrorNode(c); found != nil {
			return found
		}
	}
	return nil
}
