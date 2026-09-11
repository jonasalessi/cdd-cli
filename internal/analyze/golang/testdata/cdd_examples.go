// cdd_examples.go — docs/cdd.md section 2, in Go

package app

type Examples struct{}

func (Examples) Check(a, b, c, d int) bool {
	if a > b && c < d { // code_branch 1, condition 2
		return true
	}
	return false
}

func (Examples) IfElse(x int) int {
	if x > 0 { // code_branch 1
		return 1
	} else { // code_branch 1 — the alternative is a block
		return 2
	}
}

// unit Examples: struct, code_branch 3, condition 2, local_variable 0
