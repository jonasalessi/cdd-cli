package typescript

import (
	ts "github.com/tree-sitter/go-tree-sitter"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// The logical operators that make a Boolean clause.
const (
	opAnd           = "&&"
	opOr            = "||"
	opNullish       = "??"
	opAndAssign     = "&&="
	opOrAssign      = "||="
	opNullishAssign = "??="
	opNot           = "!"
)

// counter accumulates the raw ICP counts of one unit while its subtree is
// walked. It counts every metric, enabled or not: the pipeline drops the
// ones the configuration disables.
type counter struct {
	g      *grammar
	src    []byte
	counts map[config.MetricID]int
	// consumed holds the logical binary expressions already folded into an
	// enclosing clause chain, so a nested `&&` is never counted twice.
	consumed map[uintptr]bool
	// skipDeclarator is the unit's own declarator, which is not one of its
	// local variables; skipLambda is the unit's own function body, which is
	// not one of its lambdas.
	skipDeclarator uintptr
	skipLambda     uintptr
}

// newCounter returns a counter for the unit rooted at d.
func newCounter(g *grammar, src []byte, d *unitDecl) *counter {
	c := &counter{
		g:        g,
		src:      src,
		counts:   zeroCounts(),
		consumed: map[uintptr]bool{},
	}
	if g.kindOf(&d.node) == kindVariableDeclarator {
		c.skipDeclarator = d.node.Id()
	}
	if d.body != nil {
		c.skipLambda = d.body.Id()
	}
	return c
}

// zeroCounts returns a map holding every metric at zero.
func zeroCounts() map[config.MetricID]int {
	counts := make(map[config.MetricID]int, len(config.Metrics()))
	for _, m := range config.Metrics() {
		counts[m] = 0
	}
	return counts
}

// visit is the walk callback; it always descends, because a unit owns every
// construct nested inside it, methods and callbacks included.
func (c *counter) visit(n *ts.Node) bool {
	k := c.g.kindOf(n)
	if !c.countControlFlow(k, n) {
		c.countDeclaration(k, n)
	}
	return true
}

// countControlFlow charges the branch, condition and exception metrics,
// reporting whether k was one of theirs.
func (c *counter) countControlFlow(k kind, n *ts.Node) bool {
	switch k {
	case kindIfStatement, kindSwitchCase, kindTernaryExpression, kindForStatement,
		kindForInStatement, kindWhileStatement, kindDoStatement, kindOptionalChain:
		c.counts[config.MetricCodeBranch]++
	case kindElseClause:
		c.countElse(n)
	case kindTryStatement, kindCatchClause, kindFinallyClause:
		c.counts[config.MetricExceptionHandling]++
	case kindBinaryExpression, kindAugmentedAssignmentExpression:
		c.countCondition(k, n)
	default:
		return false
	}
	return true
}

// countDeclaration charges the inheritance, local-variable and lambda
// metrics.
//
// A local variable is one declarator, so `const {a, b} = x` is one and
// `let b = 2, c = 3` is two. A `for` initialiser holds declarators and
// counts; the binding of a `for…in` or `for…of` is not a declarator in the
// grammar and does not. Class fields count, `#private` ones included, since
// the grammar gives them all the same node; an interface's property
// signatures describe a shape rather than declare variables, and do not.
func (c *counter) countDeclaration(k kind, n *ts.Node) {
	switch k {
	case kindExtendsClause:
		c.counts[config.MetricInheritance] += namedChildrenInField(n, fieldValue)
	case kindExtendsTypeClause:
		c.counts[config.MetricInheritance] += namedChildrenInField(n, fieldType)
	case kindImplementsClause:
		c.counts[config.MetricInheritance] += int(n.NamedChildCount())
	case kindVariableDeclarator:
		if n.Id() != c.skipDeclarator {
			c.counts[config.MetricLocalVariable]++
		}
	case kindPublicFieldDefinition:
		c.counts[config.MetricLocalVariable]++
	case kindArrowFunction, kindFunctionExpression:
		if n.Id() != c.skipLambda {
			c.counts[config.MetricLambda]++
		}
	}
}

// countElse charges an `else`, unless it is the `else` of an `else if`:
// that `if` already charged itself, so `if / else if / else` is 3 and not 4.
func (c *counter) countElse(n *ts.Node) {
	if body := firstNamedChild(n); body != nil && c.g.kindOf(body) == kindIfStatement {
		return
	}
	c.counts[config.MetricCodeBranch]++
}

// countCondition charges one ICP per Boolean clause, which is the rule
// docs/cdd.md states: `if (a > b && c < d)` is 3 ICPs, "1 for the if and 1
// for each Boolean condition". The clauses of a chain are its leaf
// operands, so `a && b` is 2, `a && b || c` is 3, and a plain `if (x > 1)`
// has no logical operator and adds no condition at all.
func (c *counter) countCondition(k kind, n *ts.Node) {
	if c.consumed[n.Id()] {
		return
	}
	if k == kindAugmentedAssignmentExpression {
		c.countLogicalAssign(n)
		return
	}
	if !c.isLogical(n) {
		return
	}
	c.counts[config.MetricCondition] += c.clauses(n)
}

// countLogicalAssign charges `a &&= b`, `a ||= b` and `a ??= b`, which are
// sugar for `a = a && b`: the left-hand side is one clause and the
// right-hand side contributes its own.
func (c *counter) countLogicalAssign(n *ts.Node) {
	switch text(n.ChildByFieldId(c.g.fields.operator), c.src) {
	case opAndAssign, opOrAssign, opNullishAssign:
		c.counts[config.MetricCondition] += 1 + c.clauses(n.ChildByFieldId(c.g.fields.right))
	}
}

// clauses returns the number of Boolean clauses of the expression rooted at
// n, flattening the chain through nested logical operators, parentheses and
// `!`. Flattening through `!` follows De Morgan: `!(a || b) && x` is
// `!a && !b && x`, three clauses, not four. Every logical node it folds in
// is marked consumed so the walk does not count it again.
func (c *counter) clauses(n *ts.Node) int {
	if n == nil {
		return 0
	}
	switch c.g.kindOf(n) {
	case kindParenthesizedExpression:
		if inner := firstNamedChild(n); inner != nil {
			return c.clauses(inner)
		}
	case kindUnaryExpression:
		if text(n.ChildByFieldId(c.g.fields.operator), c.src) == opNot {
			return c.clauses(n.ChildByFieldId(c.g.fields.argument))
		}
	case kindBinaryExpression:
		if c.isLogical(n) {
			c.consumed[n.Id()] = true
			return c.clauses(n.ChildByFieldId(c.g.fields.left)) +
				c.clauses(n.ChildByFieldId(c.g.fields.right))
		}
	}
	return 1
}

// isLogical reports whether n is a binary expression whose operator joins
// Boolean clauses.
func (c *counter) isLogical(n *ts.Node) bool {
	switch text(n.ChildByFieldId(c.g.fields.operator), c.src) {
	case opAnd, opOr, opNullish:
		return true
	default:
		return false
	}
}
