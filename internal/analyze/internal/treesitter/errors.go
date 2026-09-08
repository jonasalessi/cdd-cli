package treesitter

import (
	"strconv"

	ts "github.com/tree-sitter/go-tree-sitter"
)

// SyntaxError opens the warning a file that does not parse produces; the
// position of the first error follows it.
const SyntaxError = "syntax error"

// FirstErrorNode returns the first ERROR or MISSING node in n's subtree in
// document order, nil when the subtree is clean.
func FirstErrorNode(n *ts.Node) *ts.Node {
	if n.IsError() || n.IsMissing() {
		return n
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || !c.HasError() {
			continue
		}
		if found := FirstErrorNode(c); found != nil {
			return found
		}
	}
	return nil
}

// SyntaxWarning names the position of the first error or missing node,
// 1-based, falling back to 1:1 when the root is flagged but no such node
// exists. The path stays out of it: every caller already attaches the
// warning to the file it came from.
func SyntaxWarning(root *ts.Node) string {
	line, col := 1, 1
	if n := FirstErrorNode(root); n != nil {
		line, col = Position(n)
	}
	return SyntaxError + " at " + strconv.Itoa(line) + ":" + strconv.Itoa(col)
}
