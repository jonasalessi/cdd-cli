package kotlin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// TestDocExamples pins the worked rules of docs/cdd.md section 2 against
// cdd_examples.kt (TC-B1, TC-B2).
func TestDocExamples(t *testing.T) {
	res := analyzeFixture(t, "cdd_examples.kt")
	require.Empty(t, res.Warnings)

	check := unitNamed(t, res, "check")
	requireCount(t, check, config.MetricCodeBranch, 1)
	requireCount(t, check, config.MetricCondition, 2)

	ifElse := unitNamed(t, res, "ifElse")
	requireCount(t, ifElse, config.MetricCodeBranch, 2)
	requireCount(t, ifElse, config.MetricCondition, 0)
}

// TestBranches pins the code_branch and condition totals of branches.kt,
// one function per rule (TC-B3 … TC-B11, TC-B14 … TC-B16).
func TestBranches(t *testing.T) {
	res := analyzeFixture(t, "branches.kt")
	require.Empty(t, res.Warnings)
	cases := []struct {
		unit                string
		branches, condition int
	}{
		{"describe", 2, 0},    // TC-B4: `2, 3 ->` is one entry; `else` is none
		{"chain", 4, 0},       // TC-B3: three ifs and the final else, not six
		{"safe", 2, 2},        // TC-B7: two `?.`, two clauses of `?:`
		{"loops", 4, 0},       // TC-B11: for, for, while, do…while; `!!` is none
		{"subjectless", 2, 0}, // TC-B5: `a > 1` is a comparison, not a clause
		{"Holder", 2, 0},      // TC-B6: `when` as a body and inside a lambda
		{"deep", 3, 0},        // TC-B8: `a?.b?.c?.d`
		{"plain", 0, 0},       // TC-B8: `.` never counts
		{"let", 1, 0},         // TC-B9
		{"bang", 0, 0},        // TC-B10
		{"guardedIf", 2, 2},   // TC-B14
		{"elvisReturn", 0, 2}, // TC-B15: `return` is not a branch
		{"elvisThrow", 0, 2},  // TC-B15: `throw` is not a branch
		{"tryValue", 0, 0},    // TC-B16: `try` is not a branch
	}
	for _, c := range cases {
		t.Run(c.unit, func(t *testing.T) {
			u := unitNamed(t, res, c.unit)
			requireCount(t, u, config.MetricCodeBranch, c.branches)
			requireCount(t, u, config.MetricCondition, c.condition)
		})
	}
}

// TestConditionClauses pins the clause counting of conditions.kt, one
// function per shape (TC-B12, TC-B13).
func TestConditionClauses(t *testing.T) {
	res := analyzeFixture(t, "conditions.kt")
	require.Empty(t, res.Warnings)
	cases := map[string]int{
		"and2":    2,
		"andOr3":  3,
		"parens4": 4, // `(a && b) || !(c || d)`: parentheses and `!` are transparent
		"not0":    0,
		"eq0":     0,
		"elvis2":  2,
		"elvis3":  3,
		"args4":   4, // TC-B13: two chains in one argument list, nothing counted twice
	}
	for unit, want := range cases {
		t.Run(unit, func(t *testing.T) {
			u := unitNamed(t, res, unit)
			requireCount(t, u, config.MetricCondition, want)
			requireCount(t, u, config.MetricCodeBranch, 0)
		})
	}
}

// TestElseAfterAComment keeps `else // why\n {` on the branch rather than
// on the comment, which the grammar may place between the keyword and the
// block.
func TestElseAfterAComment(t *testing.T) {
	res := analyzeSource(t, "fun f(a: Boolean) {\n    if (a) {\n    } else // why\n    {\n    }\n}\n")
	requireCount(t, unitNamed(t, res, "f"), config.MetricCodeBranch, 2)
}
