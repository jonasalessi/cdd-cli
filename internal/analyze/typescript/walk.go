package typescript

import (
	ts "github.com/tree-sitter/go-tree-sitter"
)

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
