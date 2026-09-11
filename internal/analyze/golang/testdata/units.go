package app

import "sync"

type A struct{}

type B interface{}

type C int

type D = A

type E func(int) error

func (g G) Early() {}

type (
	F struct{}
	G int
)

func (a A) M() {}

func (c *C) M() {}

func (H) M() {}

func (h H) N() {}

func Free() {}

func init() {}

var v sync.Mutex

const k = 1

// units in order: A struct, B interface, C type, D type, E type, F struct,
// G type, H methods, Free func, init func — ten.
// A sits at the name `A`, F and G at their names inside the group and not at
// the shared `type` keyword; `Early` precedes the group and still bills to G;
// the `methods` unit H sits at the `func` of `func (H) M()`, the first method
// of a receiver with no TypeSpec here.
// `var v sync.Mutex` and `const k = 1` are not units, so `sync` is charged to
// no unit and neither declaration adds a local_variable anywhere.
