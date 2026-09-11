// coupling_dot.go — a dot import charges every unit

package billing

import (
	. "math"

	"example.com/app/shared"
)

type Invoice struct{}

func (Invoice) Round(v float64) float64 {
	return Floor(v) * shared.Tax() // internal_coupling 1 (shared)
}

type Plain struct{}

// unit Invoice: 1 / 0 / 1 — shared, plus `math` through the dot import
// unit Plain: 0 / 0 / 1 — the dot import charges every unit, because a dot
//   name is indistinguishable from a local
