package analyze

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMatcher builds a matcher a test can use inline.
func newMatcher(t *testing.T, include, exclude []string) *Matcher {
	t.Helper()
	m, err := NewMatcher(include, exclude)
	require.NoError(t, err)
	return m
}

func TestMatcherGlobs(t *testing.T) {
	tests := []struct {
		glob  string
		path  string
		match bool
	}{
		{glob: "**/*.test.ts", path: "src/a/b.test.ts", match: true},
		{glob: "**/*.test.ts", path: "b.test.ts", match: true},
		{glob: "**/*.test.ts", path: "src/b.testxts", match: false},
		{glob: "vendor/**", path: "vendor/x/y.go", match: true},
		{glob: "vendor/**", path: "vendor/y.go", match: true},
		{glob: "vendor/**", path: "src/vendor.go", match: false},
		{glob: "vendor/**", path: "src/vendor/y.go", match: false},
		{glob: "**/node_modules/**", path: "node_modules/left-pad/index.ts", match: true},
		{glob: "**/node_modules/**", path: "web/node_modules/x/y.ts", match: true},
		{glob: "**/node_modules/**", path: "web/node_modules.ts", match: false},
		{glob: "src/*.ts", path: "src/a.ts", match: true},
		// A single star stops at the separator; a double star crosses it.
		{glob: "src/*.ts", path: "src/a/b.ts", match: false},
		{glob: "src/**/*.ts", path: "src/a.ts", match: true},
		{glob: "src/**/*.ts", path: "src/a/b/c.ts", match: true},
		{glob: "a?c.ts", path: "abc.ts", match: true},
		{glob: "a?c.ts", path: "ac.ts", match: false},
		{glob: "a?c.ts", path: "a/c.ts", match: false},
		{glob: "*.[jt]s", path: "main.ts", match: true},
		{glob: "*.[jt]s", path: "main.js", match: true},
		{glob: "*.[jt]s", path: "main.rs", match: false},
		{glob: "*.[!jt]s", path: "main.rs", match: true},
		{glob: "*.[!jt]s", path: "main.ts", match: false},
		{glob: "main.ts", path: "main.ts", match: true},
		{glob: "main.ts", path: "src/main.ts", match: false},
		// Every other character is quoted, so the dot is literal.
		{glob: "a.b", path: "axb", match: false},
		{glob: "glob:vendor/**", path: "vendor/x.go", match: true},
	}
	for _, tt := range tests {
		t.Run(tt.glob+" vs "+tt.path, func(t *testing.T) {
			m := newMatcher(t, []string{tt.glob}, nil)
			assert.Equal(t, tt.match, m.Match(tt.path))
		})
	}
}

func TestMatcherRegexEntriesAreNotGlobs(t *testing.T) {
	m := newMatcher(t, []string{`regex:.*\.test\.ts$`}, nil)
	assert.True(t, m.Match("src/a/b.test.ts"))
	assert.False(t, m.Match("src/a/b.ts"))

	// The same text read as a glob matches nothing, which is what tells the
	// two syntaxes apart.
	glob := newMatcher(t, []string{`.*\.test\.ts$`}, nil)
	assert.False(t, glob.Match("src/a/b.test.ts"))
}

func TestMatcherRegexIsNotAnchored(t *testing.T) {
	m := newMatcher(t, []string{"regex:adapters"}, nil)
	assert.True(t, m.Match("src/adapters/http.ts"))
}

func TestMatcherEmptyIncludeIncludesEverything(t *testing.T) {
	m := newMatcher(t, nil, nil)
	assert.True(t, m.Match("anything/at/all.ts"))
	assert.True(t, m.Match("main.go"))
}

func TestMatcherExcludeWinsOverInclude(t *testing.T) {
	m := newMatcher(t, []string{"src/**"}, []string{"**/*.test.ts"})
	assert.True(t, m.Match("src/order.ts"))
	assert.False(t, m.Match("src/order.test.ts"))
	assert.False(t, m.Match("lib/order.ts"), "outside every include pattern")
}

func TestMatcherExcludeAloneKeepsTheRest(t *testing.T) {
	m := newMatcher(t, nil, []string{"vendor/**", "regex:_generated\\.go$"})
	assert.False(t, m.Match("vendor/x/y.go"))
	assert.False(t, m.Match("src/api_generated.go"))
	assert.True(t, m.Match("src/api.go"))
}

func TestMatcherIncludeMatchesAnyEntry(t *testing.T) {
	m := newMatcher(t, []string{"src/**", "lib/**"}, nil)
	assert.True(t, m.Match("src/a.ts"))
	assert.True(t, m.Match("lib/a.ts"))
	assert.False(t, m.Match("test/a.ts"))
}

func TestNewMatcherRejectsInvalidPatterns(t *testing.T) {
	tests := map[string]struct {
		include []string
		exclude []string
		message string
	}{
		"invalid include regex": {
			include: []string{"regex:("},
			message: "include: invalid regex",
		},
		"invalid exclude regex": {
			exclude: []string{"regex:["},
			message: "exclude: invalid regex",
		},
		"unterminated class": {
			exclude: []string{"*.[jt"},
			message: "unterminated character class",
		},
		"empty class": {
			exclude: []string{"*.[]"},
			message: "unterminated character class",
		},
		"empty entry": {
			include: []string{"   "},
			message: "include: empty pattern",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewMatcher(tt.include, tt.exclude)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestNewMatcherReportsAGlobThatCompilesToABadRegex(t *testing.T) {
	_, err := NewMatcher(nil, []string{"[^]"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid glob")
}
