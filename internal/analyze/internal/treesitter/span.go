package treesitter

import (
	ts "github.com/tree-sitter/go-tree-sitter"
)

// Span is one source range the way analyze.Occurrence carries it: 1-based
// line and column of the first character, and the line and column just past
// the last.
type Span struct {
	Line, Col       int
	EndLine, EndCol int
}

// SpanOf returns n's range. Tree-sitter rows and columns are 0-based and its
// end position is already exclusive, so only the origin has to move.
func SpanOf(n *ts.Node) Span {
	start, end := n.StartPosition(), n.EndPosition()
	return Span{
		Line:    int(start.Row) + 1,
		Col:     int(start.Column) + 1,
		EndLine: int(end.Row) + 1,
		EndCol:  int(end.Column) + 1,
	}
}

// Position returns n's 1-based line and column.
func Position(n *ts.Node) (line, col int) {
	s := SpanOf(n)
	return s.Line, s.Col
}
