package golang

import (
	"go/ast"
	"go/token"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/analyze/internal/treesitter"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// counter accumulates the raw ICP counts of one unit while the subtrees it
// owns are walked. It counts every metric, enabled or not: the pipeline
// drops the ones the configuration disables (FR-4).
//
// Every count is charged through charge or chargeSpan, which record where it
// came from, so a unit's Counts is always the sum of its Occurrences' Count,
// metric by metric.
type counter struct {
	fset   *token.FileSet
	counts map[config.MetricID]int
	// occurrences locate every charge, in the order it was made. ast.Inspect
	// walks in pre-order, so they come out in source order except for the
	// leaf clauses flattened out of a chain; sortedOccurrences orders them.
	occurrences []analyze.Occurrence
	// consumed holds the logical expressions already folded into an
	// enclosing clause chain, so a nested `&&` is never counted twice.
	consumed map[ast.Node]bool
}

// newCounter returns a counter for the subtrees of one unit.
func newCounter(fset *token.FileSet) *counter {
	return &counter{
		fset:     fset,
		counts:   zeroCounts(),
		consumed: map[ast.Node]bool{},
	}
}

// zeroCounts returns a map holding every metric at zero. A unit always
// carries a key for every metric, enabled or not: the pipeline drops the
// ones the configuration disables (FR-4).
func zeroCounts() map[config.MetricID]int {
	counts := make(map[config.MetricID]int, len(config.Metrics()))
	for _, m := range config.Metrics() {
		counts[m] = 0
	}
	return counts
}

// measureUnit counts the ICPs of one unit over every subtree billed to it —
// the type declaration plus its methods, or the function itself — and
// reports the unit the pipeline consumes (FR-4).
func measureUnit(fset *token.FileSet, d unitDecl) analyze.Unit {
	c := newCounter(fset)
	for _, n := range d.nodes {
		ast.Inspect(n, c.visit)
	}
	return analyze.Unit{
		Name:        d.name,
		Kind:        d.kind,
		Line:        d.line,
		Col:         d.col,
		Counts:      c.counts,
		Occurrences: c.sortedOccurrences(),
	}
}

// charge adds one point of metric to the unit and records n's range as where
// it comes from. Every Go construct is worth one point: the language has no
// form that folds two decisions into one node.
func (c *counter) charge(metric config.MetricID, n ast.Node) {
	c.chargeSpan(metric, c.span(n))
}

// chargeSpan is charge for a range the caller computed itself, which is how
// a charge that points at part of a node is located.
func (c *counter) chargeSpan(metric config.MetricID, s treesitter.Span) {
	c.counts[metric]++
	c.occurrences = append(c.occurrences, analyze.Occurrence{
		Metric:  metric,
		Line:    s.Line,
		Col:     s.Col,
		EndLine: s.EndLine,
		EndCol:  s.EndCol,
		Count:   1,
	})
}

// span returns n's range the way analyze.Occurrence carries it. go/token
// positions are 1-based with byte columns and End is already exclusive, so
// they are the contract treesitter.SpanOf produces and spans compare across
// languages.
func (c *counter) span(n ast.Node) treesitter.Span {
	start, end := c.fset.Position(n.Pos()), c.fset.Position(n.End())
	return treesitter.Span{
		Line:    start.Line,
		Col:     start.Column,
		EndLine: end.Line,
		EndCol:  end.Column,
	}
}

// sortedOccurrences returns the unit's occurrences in source order.
func (c *counter) sortedOccurrences() []analyze.Occurrence {
	treesitter.SortOccurrences(c.occurrences)
	return c.occurrences
}

// visit is the ast.Inspect callback; it always descends, because a unit owns
// every construct nested inside it, func literals and local types included.
// The two halves of the walk are independent: a node is either a control
// flow construct or a declaration, never both.
func (c *counter) visit(n ast.Node) bool {
	c.countControlFlow(n)
	c.countDeclaration(n)
	return true
}

// countControlFlow charges the branch and condition metrics.
//
// An `if` is one branch and its `else` another, unless that `else` opens
// another `if`, which charges itself (FR-6). A `case` of a switch or of a
// type switch and a `case` of a select are one branch each however many
// values they list, and the `default` arm is none. Each loop is one branch:
// the reader still has to decide whether to go round again, in all four
// shapes of `for` and over a `range`. A `return`, a `break`, a `continue`, a
// `goto`, a `fallthrough`, a label, a `go`, a `defer`, a type assertion,
// `panic` and `recover` are not branches, and Go has no handler construct,
// so exception_handling is never charged.
func (c *counter) countControlFlow(n ast.Node) {
	switch n := n.(type) {
	case *ast.IfStmt:
		c.charge(config.MetricCodeBranch, n)
		c.countElse(n)
	case *ast.CaseClause:
		c.countArm(n, n.List != nil)
	case *ast.CommClause:
		c.countArm(n, n.Comm != nil)
	case *ast.ForStmt, *ast.RangeStmt:
		c.charge(config.MetricCodeBranch, n)
	case *ast.BinaryExpr:
		c.countCondition(n)
	}
}

// countDeclaration charges the metrics a declaration carries — embedding,
// locals and func literals. Their rules land with the tasks that own them;
// the seam is here so the walk is split by rule from the start and neither
// half grows past what one reader can hold.
func (c *counter) countDeclaration(_ ast.Node) {}

// countArm charges one arm of a switch, a type switch or a select when the
// arm tests something. A `default` tests nothing and is 0: it is where every
// value not matched above falls, not a decision of its own.
func (c *counter) countArm(n ast.Node, tests bool) {
	if tests {
		c.charge(config.MetricCodeBranch, n)
	}
}

// countElse charges the alternative of an `if` only when it is a block: an
// `else if` is an IfStmt that charges itself, so `if / else if / else` is 3
// and not 4 (FR-6). The occurrence spans the block, since go/ast keeps no
// position for the `else` keyword.
func (c *counter) countElse(n *ast.IfStmt) {
	if block, ok := n.Else.(*ast.BlockStmt); ok {
		c.charge(config.MetricCodeBranch, block)
	}
}

// countCondition charges one ICP per Boolean clause, which is the rule
// docs/cdd.md states: `if a > b && c < d` is 3 ICPs, "1 for the if and 1 for
// each Boolean condition". The clauses of a chain are its leaf operands, so
// `a && b` is 2 and `a && b || c` is 3, while a plain `if x > 1` joins
// nothing and adds no condition at all (FR-5).
func (c *counter) countCondition(n *ast.BinaryExpr) {
	if c.consumed[n] || !isLogical(n) {
		return
	}
	for _, clause := range c.clauses(n, nil) {
		c.charge(config.MetricCondition, clause)
	}
}

// clauses appends the Boolean clauses of the expression rooted at n to out
// and returns it. The clauses of a chain are its leaf operands: the walk
// flattens through nested logical operators, parentheses and `!`, so what
// lands in out is the operand itself, `a` rather than `!a`. Flattening
// through `!` follows De Morgan: `!(a || b) && x` is `!a && !b && x`, three
// clauses, not four. Every logical node it folds in is marked consumed so
// the walk does not count it again.
func (c *counter) clauses(n ast.Expr, out []ast.Expr) []ast.Expr {
	switch e := n.(type) {
	case *ast.ParenExpr:
		return c.clauses(e.X, out)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return c.clauses(e.X, out)
		}
	case *ast.BinaryExpr:
		if isLogical(e) {
			c.consumed[e] = true
			return c.clauses(e.Y, c.clauses(e.X, out))
		}
	}
	return append(out, n)
}

// isLogical reports whether a binary expression joins Boolean clauses. Go's
// short-circuit pair is the whole list: `&`, `|`, `^` and `&^` are
// arithmetic on bits and never clauses.
func isLogical(n *ast.BinaryExpr) bool {
	return n.Op == token.LAND || n.Op == token.LOR
}
