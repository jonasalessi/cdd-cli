// lambdas.go — func literals

package app

import "sort"

type Lambdas struct{}

func (Lambdas) Wire(xs []int) {
	double := func(x int) int { // lambda 1, local_variable 1 (double)
		return x * 2
	}
	sort.Slice(xs, func(i, j int) bool { // lambda 1, stdlib_coupling 1 (sort)
		return xs[i] < xs[j]
	})
	go func() {}()    // lambda 1
	defer func() {}() // lambda 1
	_ = double
}

func Value(l Lambdas) func([]int) {
	return l.Wire // lambda 0 — a method value is not a func literal
}

// unit Lambdas: struct, lambda 4, local_variable 1, stdlib_coupling 1
// unit Value: func, lambda 0, local_variable 0
