// branches.go — code_branch

package app

type Branches struct{}

func (Branches) Switch(x int) string {
	switch x {
	case 1: // code_branch 1 — an arm of one value
		return "one"
	case 2, 3: // code_branch 1 — one arm, two values
		return "few"
	default: // code_branch 0
		return "many"
	}
}

func (Branches) TypeSwitch(v any) string {
	switch t := v.(type) { // local_variable 0 — the type-switch guard
	case int: // code_branch 1
		_ = t
	case string, []byte: // code_branch 1 — one arm, two types
		_ = t
	default: // code_branch 0
	}
	return ""
}

func (Branches) Select(a, b chan int) {
	select {
	case x := <-a: // code_branch 1, local_variable 1 (x)
		_ = x
	case <-b: // code_branch 1
	default: // code_branch 0
	}
}

func (Branches) Chain(a, b, c bool) int {
	if a { // code_branch 1
		return 1
	} else if b { // code_branch 1 — the if; its else is an if, so no point
		return 2
	} else if c { // code_branch 1
		return 3
	} else { // code_branch 1 — the alternative is a block
		return 4
	}
}

func (Branches) Loops(xs []int, flag bool) {
	for i := 0; i < 3; i++ { // code_branch 1, local_variable 1 (i)
		noop(i)
	}
	for _, x := range xs { // code_branch 1, local_variable 1 (x); `_` is 0
		noop(x)
	}
	for flag { // code_branch 1
		break
	}
	for { // code_branch 1
		break
	}
}

func noop(_ ...int) {}

// unit Branches: struct, code_branch 14, condition 0, local_variable 3
// unit noop: func, every metric 0 — a variadic `_` parameter is not a variable
