package typescript

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsBuiltin covers the Node.js built-in table on its own: the "node:"
// prefix, the bare names, their submodules, and the look-alikes that must
// stay external.
func TestIsBuiltin(t *testing.T) {
	cases := []struct {
		spec string
		want bool
	}{
		{"fs", true},
		{"node:fs", true},
		{"fs/promises", true},
		{"node:fs/promises", true},
		{"path/posix", true},
		{"stream/web", true},
		{"timers/promises", true},
		{"util/types", true},
		{"node:test", true},
		{"node:sqlite", true},
		{"console", true},
		{"node-fetch", false},
		{"fsx", false},
		{"@types/node", false},
		{"lodash/fp", false},
		{"react", false},
		{"./repo", false},
		{"", false},
	}
	for _, c := range cases {
		t.Run(c.spec, func(t *testing.T) {
			require.Equal(t, c.want, isBuiltin(c.spec))
		})
	}
}
