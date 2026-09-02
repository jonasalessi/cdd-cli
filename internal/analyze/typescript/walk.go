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

// namedChildrenInField returns n's named children that sit in the given
// field, which separates `extends Base<T>`'s one parent from its type
// arguments. An empty field takes every named child.
func namedChildrenInField(n *ts.Node, field string) []ts.Node {
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

// srcSpan is one source range the way analyze.Occurrence carries it: 1-based
// line and column of the first character, and the line and column just past
// the last.
type srcSpan struct {
	line, col       int
	endLine, endCol int
}

// spanOf returns n's range. Tree-sitter rows and columns are 0-based and its
// end position is already exclusive, so only the origin has to move.
func spanOf(n *ts.Node) srcSpan {
	start, end := n.StartPosition(), n.EndPosition()
	return srcSpan{
		line:    int(start.Row) + 1,
		col:     int(start.Column) + 1,
		endLine: int(end.Row) + 1,
		endCol:  int(end.Column) + 1,
	}
}

// position returns n's 1-based line and column.
func position(n *ts.Node) (line, col int) {
	s := spanOf(n)
	return s.line, s.col
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
