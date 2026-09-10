package java

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// inMethod wraps a method body in the class Wrapper, so a rule can be pinned
// on the shortest source that shows it.
func inMethod(body string) string {
	return "class Wrapper {\n    void f(boolean a, boolean b, boolean c, int x, Object o) {\n" +
		body + "\n    }\n}\n"
}

// branchesOf returns the code_branch occurrences of a unit, in source order.
func branchesOf(u analyze.Unit) []analyze.Occurrence {
	out := make([]analyze.Occurrence, 0, len(u.Occurrences))
	for _, o := range u.Occurrences {
		if o.Metric == config.MetricCodeBranch {
			out = append(out, o)
		}
	}
	return out
}

// TestDocExamples pins the worked rules of docs/cdd.md section 2 against
// cdd_examples.java (TC-B1, TC-B2). Exceptions land with T6, so the unit's
// exception_handling is still zero here.
func TestDocExamples(t *testing.T) {
	res := analyzeFixture(t, "cdd_examples.java")
	require.Empty(t, res.Warnings)

	examples := unitNamed(t, res, "Examples")
	requireCount(t, examples, config.MetricCodeBranch, 3)
	requireCount(t, examples, config.MetricCondition, 2)
	requireCount(t, examples, config.MetricExceptionHandling, 0)
	requireCount(t, examples, config.MetricLocalVariable, 0)
}

// TestBranchesFixture pins the code_branch total of branches.java (TC-B12).
// The locals land with T6, so local_variable is still zero.
func TestBranchesFixture(t *testing.T) {
	res := analyzeFixture(t, "branches.java")
	require.Empty(t, res.Warnings)

	branches := unitNamed(t, res, "Branches")
	requireCount(t, branches, config.MetricCodeBranch, 13)
	requireCount(t, branches, config.MetricCondition, 0)
	requireCount(t, branches, config.MetricLocalVariable, 0)
}

// TestConditionsFixture pins the condition total of conditions.java
// (TC-B13): 2 + 3 + 3 + 0 + 0.
func TestConditionsFixture(t *testing.T) {
	res := analyzeFixture(t, "conditions.java")
	require.Empty(t, res.Warnings)

	conditions := unitNamed(t, res, "Conditions")
	requireCount(t, conditions, config.MetricCondition, 8)
	requireCount(t, conditions, config.MetricCodeBranch, 0)
}

// TestIfChains pins FR-9: an `else` is a branch of its own unless it opens
// another `if` (TC-B2, TC-B3, TC-B4).
func TestIfChains(t *testing.T) {
	cases := map[string]int{
		"if (a) { }":                                          1,
		"if (a) { } else { }":                                 2,
		"if (a) { } else if (b) { }":                          2,
		"if (a) { } else if (b) { } else { }":                 3,
		"if (a) { } else if (b) { } else if (c) { }":          3,
		"if (a) { if (b) { } else { } } else { }":             4,
		"if (a) { } else if (b) { } else if (c) { } else { }": 4,
	}
	for src, want := range cases {
		t.Run(src, func(t *testing.T) {
			res := analyzeSource(t, inMethod("        "+src))
			requireCount(t, unitNamed(t, res, "Wrapper"), config.MetricCodeBranch, want)
		})
	}
}

// TestElseOccurrenceStartsAtTheKeyword pins where an `else` charge points
// (TC-B3): at the keyword, not back at the `if` the statement starts with.
func TestElseOccurrenceStartsAtTheKeyword(t *testing.T) {
	res := analyzeSource(t, inMethod("        if (a) {\n        } else {\n        }"))
	got := branchesOf(unitNamed(t, res, "Wrapper"))
	require.Len(t, got, 2)
	require.Equal(t, 3, got[0].Line, "the if starts the statement")
	require.Equal(t, 9, got[0].Col)
	require.Equal(t, 4, got[1].Line, "the else charge starts at the else keyword")
	require.Equal(t, 11, got[1].Col)
	require.Equal(t, 5, got[1].EndLine, "and ends with the branch")
}

// TestSwitchArms pins FR-10 over every shape the grammar produces. An
// old-style arm is spread over one group per label, so a fallthrough is a
// run of label-only groups ending in the group that carries the code; the
// arm is one branch when any label of that run tests a value (TC-B5, TC-B6,
// TC-B7, TC-B8, TC-B9).
func TestSwitchArms(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		branches int
	}{
		{"one label", "switch (x) { case 1: g(); }", 1},
		{"fallthrough", "switch (x) { case 1: case 2: g(); }", 1},
		{"fallthrough with a comment", "switch (x) { case 1: /* falls */ case 2: g(); }", 1},
		{"two arms", "switch (x) { case 1: g(); case 2: g(); }", 2},
		{"default only", "switch (x) { default: g(); }", 0},
		{"case then default", "switch (x) { case 1: default: g(); }", 1},
		{"default then case", "switch (x) { default: case 1: g(); }", 1},
		{"arm then default", "switch (x) { case 1: g(); default: g(); }", 1},
		{"arrow", "switch (x) { case 1 -> g(); case 2, 3 -> g(); default -> g(); }", 2},
		{"arrow as expression", "int y = switch (x) { case 1 -> 1; case 2, 3 -> 2; default -> 3; };", 2},
		{"pattern arms", "switch (o) { case String s -> g(); case Integer i -> g(); default -> g(); }", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := analyzeSource(t, inMethod("        "+c.body))
			u := unitNamed(t, res, "Wrapper")
			requireCount(t, u, config.MetricCodeBranch, c.branches)
			requireCount(t, u, config.MetricLocalVariable, 0)
		})
	}
}

// TestFallthroughArmStartsAtItsFirstLabel pins the range of a fallthrough
// arm (TC-B5): it covers the labels a reader has to read together, from the
// first `case` to the end of the statements they share.
func TestFallthroughArmStartsAtItsFirstLabel(t *testing.T) {
	res := analyzeFixture(t, "branches.java")
	got := branchesOf(unitNamed(t, res, "Branches"))
	require.Equal(t, 8, got[0].Line, "the arm of `case 1:`")
	require.Equal(t, 10, got[1].Line, "the fallthrough arm starts at `case 2:`")
	require.Equal(t, 12, got[1].EndLine, "and ends with the statements of `case 3:`")
}

// TestNonBranchingStatements pins what is not a branch (TC-B16, TC-B17). The
// `try` is exception_handling, which lands with T6.
func TestNonBranchingStatements(t *testing.T) {
	cases := map[string]int{
		"return;":                                       0,
		"if (o instanceof String s) { }":                1,
		"boolean t = o instanceof String;":              0,
		"assert a;":                                     0,
		"throw new IllegalStateException();":            0,
		"try { g(); } catch (Exception e) { }":          0,
		"try { while (a) { } } catch (Exception e) { }": 1,
		"outer: while (a) { break outer; }":             1,
	}
	for src, want := range cases {
		t.Run(src, func(t *testing.T) {
			res := analyzeSource(t, inMethod("        "+src))
			requireCount(t, unitNamed(t, res, "Wrapper"), config.MetricCodeBranch, want)
		})
	}
}

// TestTernaries pins FR-10's ternary rule (TC-B10): each `? :` is one
// decision, so a nested one is two.
func TestTernaries(t *testing.T) {
	cases := map[string]int{
		"int y = a ? 1 : 2;":         1,
		"int y = a ? 1 : b ? 2 : 3;": 2,
	}
	for src, want := range cases {
		t.Run(src, func(t *testing.T) {
			res := analyzeSource(t, inMethod("        "+src))
			requireCount(t, unitNamed(t, res, "Wrapper"), config.MetricCodeBranch, want)
		})
	}
}

// TestLoops pins one branch per loop statement (TC-B11).
func TestLoops(t *testing.T) {
	body := "        for (int i = 0; i < 3; i++) { }\n" +
		"        for (Object e : java.util.List.of()) { }\n" +
		"        while (a) { }\n" +
		"        do { } while (a);"
	res := analyzeSource(t, inMethod(body))
	requireCount(t, unitNamed(t, res, "Wrapper"), config.MetricCodeBranch, 4)
}

// TestConditionClauses pins FR-8 shape by shape: a clause is a leaf operand
// of a chain, parentheses and `!` are transparent, and a comparison or a
// bitwise operator is no clause at all (TC-B13, TC-B14).
func TestConditionClauses(t *testing.T) {
	cases := map[string]int{
		"boolean r = a && b;":           2,
		"boolean r = a && b || c;":      3,
		"boolean r = !(a || b) && c;":   3, // De Morgan: three clauses, not four
		"boolean r = a && (b || c);":    3,
		"boolean r = !a && !b;":         2,
		"boolean r = x > 1;":            0,
		"boolean r = a;":                0,
		"int r = x & 1 | 2 ^ 3;":        0,
		"g(a && b, c || !a);":           4, // TC-B14: nothing counted twice
		"boolean r = a && b && c && a;": 4,
	}
	for src, want := range cases {
		t.Run(src, func(t *testing.T) {
			res := analyzeSource(t, inMethod("        "+src))
			requireCount(t, unitNamed(t, res, "Wrapper"), config.MetricCondition, want)
		})
	}
}

// TestGuardedIfCountsBranchAndClauses pins that an `if` and its clauses are
// charged together, and that the `if` occurrence sorts before them (TC-B15).
func TestGuardedIfCountsBranchAndClauses(t *testing.T) {
	res := analyzeSource(t, inMethod("        if (a && b) { g(); } else { g(); }"))
	u := unitNamed(t, res, "Wrapper")
	requireCount(t, u, config.MetricCodeBranch, 2)
	requireCount(t, u, config.MetricCondition, 2)
	require.Equal(t, config.MetricCodeBranch, u.Occurrences[0].Metric, "the if comes first")
	require.Equal(t, config.MetricCondition, u.Occurrences[1].Metric)
}

// TestNestedDeclarationsBillToTheUnit (TC-U5): a branch written inside a
// nested type is the top-level unit's, like everything else nested in it.
func TestNestedDeclarationsBillToTheUnit(t *testing.T) {
	src := "class Outer {\n    static class Inner {\n" +
		"        void m(int v) {\n            if (v > 0) { }\n        }\n    }\n}\n"
	res := analyzeSource(t, src)
	require.Equal(t, []string{"Outer"}, unitNames(res))
	requireCount(t, unitNamed(t, res, "Outer"), config.MetricCodeBranch, 1)
}

// TestCountsEqualTheirOccurrences pins the FR-4 invariant on the fixtures
// this task counts: a count is the number of occurrences that produced it.
func TestCountsEqualTheirOccurrences(t *testing.T) {
	for _, fixture := range []string{"cdd_examples.java", "branches.java", "conditions.java"} {
		t.Run(fixture, func(t *testing.T) {
			for _, u := range analyzeFixture(t, fixture).Units {
				charged := map[config.MetricID]int{}
				for _, o := range u.Occurrences {
					charged[o.Metric] += o.Count
				}
				for _, m := range config.Metrics() {
					require.Equal(t, u.Counts[m], charged[m], "unit %q, metric %q", u.Name, m)
				}
			}
		})
	}
}
