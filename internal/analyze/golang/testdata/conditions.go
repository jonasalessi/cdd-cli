// conditions.go — condition clauses

package app

type Conditions struct{}

func (Conditions) Both(a, b bool) bool {
	return a && b // condition 2
}

func (Conditions) Either(a, b, c bool) bool {
	return a && b || c // condition 3
}

func (Conditions) Negated(a, b, x bool) bool {
	return !(a || b) && x // condition 3 — flattened through ! and ( )
}

func (Conditions) Compare(x int) bool {
	return x > 1 // condition 0 — a comparison is not a clause
}

func (Conditions) Bits(a, b int) int {
	return a&b | 3 // condition 0 — bitwise operators are not clauses
}

// unit Conditions: struct, condition 8, code_branch 0
