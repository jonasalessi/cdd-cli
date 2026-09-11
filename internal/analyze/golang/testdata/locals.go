// locals.go — local_variable

package app

type Locals struct {
	a, b int    // local_variable 2 — one per name
	c    string // local_variable 1
}

func (l Locals) Body(o any) {
	var x, y = 1, 2    // local_variable 2
	z := x             // local_variable 1
	z, err := split(y) // local_variable 1 — z redeclares, err is new
	if err != nil {    // code_branch 1
		return
	}
	const limit = 10             // local_variable 1
	if s, ok := o.(string); ok { // code_branch 1, local_variable 2 (s, ok)
		_ = s
	}
	switch v := o.(type) { // local_variable 0 — the type-switch guard
	case int: // code_branch 1
		_ = v
	}
	_, _, _ = z, limit, l
}

func split(n int) (int, error) { // local_variable 0 — parameters and results
	return n, nil
}

func named() (n int, err error) { // local_variable 0 — named results
	return 0, nil
}

// unit Locals: struct, local_variable 10, code_branch 3
// units split, named: func, local_variable 0
