package jvm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var javaExt = []string{".java"}

// cancelAfterContext cancels its valid underlying context on the requested
// Err check, making a walk cancellation deterministic without timing.
type cancelAfterContext struct {
	context.Context
	cancel    context.CancelFunc
	remaining atomic.Int32
}

func newCancelAfterContext(parent context.Context, checks int32) *cancelAfterContext {
	ctx, cancel := context.WithCancel(parent)
	result := &cancelAfterContext{Context: ctx, cancel: cancel}
	result.remaining.Store(checks)
	return result
}

func (c *cancelAfterContext) Err() error {
	if c.remaining.Add(-1) == 0 {
		c.cancel()
	}
	return c.Context.Err()
}

// writeJVM lays out a java source declaring pkg at the given relative path.
func writeJVM(t *testing.T, root, rel, pkg string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("package "+pkg+";\n\nclass C {}\n"), 0o644))
}

func TestPrefixesReducesDeclarations(t *testing.T) {
	dir := t.TempDir()
	writeJVM(t, dir, "src/main/java/com/acme/billing/api/Invoice.java", "com.acme.billing.api")
	writeJVM(t, dir, "src/main/java/com/acme/billing/db/InvoiceRow.java", "com.acme.billing.db")
	writeJVM(t, dir, "src/main/java/com/acme/shared/Money.java", "com.acme.shared")
	got, err := Prefixes(t.Context(), dir, javaExt)
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got)
}

func TestPrefixesReadsOnlyTheGivenExtensions(t *testing.T) {
	dir := t.TempDir()
	writeJVM(t, dir, "src/main/java/com/acme/app/C.java", "com.acme.app")
	writeJVM(t, dir, "src/main/kotlin/com/other/K.kt", "com.other")
	java, err := Prefixes(t.Context(), dir, javaExt)
	require.NoError(t, err)
	kotlin, err := Prefixes(t.Context(), dir, []string{".KT"})
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.app"}, java)
	assert.Equal(t, []string{"com.other"}, kotlin, "extensions match case-insensitively")
}

func TestPrefixesMissingRoot(t *testing.T) {
	got, err := Prefixes(t.Context(), filepath.Join(t.TempDir(), "gone"), javaExt)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestPrefixesSkipsBuildOutput(t *testing.T) {
	dir := t.TempDir()
	writeJVM(t, dir, "src/main/java/com/acme/app/C.java", "com.acme.app")
	writeJVM(t, dir, "build/generated/com/vendor/gen/G.java", "com.vendor.gen")
	got, err := Prefixes(t.Context(), dir, javaExt)
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.app"}, got, "a build directory at the root is not scanned")
}

func TestPrefixesStopsAtTheFileCap(t *testing.T) {
	dir := t.TempDir()
	// One package past the cap, so the walk must stop before reaching it.
	for i := range maxFiles + 1 {
		writeJVM(t, dir, fmt.Sprintf("src/p%03d/C.java", i), fmt.Sprintf("com.acme.p%03d", i))
	}
	got, err := Prefixes(t.Context(), dir, javaExt)
	require.NoError(t, err)
	assert.NotEmpty(t, got)
	assert.LessOrEqual(t, len(got), maxPrefixes, "the prefix list is capped too")
}

func TestPrefixesCapsThePrefixList(t *testing.T) {
	dir := t.TempDir()
	for i := range maxPrefixes + 3 {
		writeJVM(t, dir, fmt.Sprintf("src/r%d/C.java", i), fmt.Sprintf("root%d.pkg", i))
	}
	got, err := Prefixes(t.Context(), dir, javaExt)
	require.NoError(t, err)
	assert.Len(t, got, maxPrefixes)
}

func TestPrefixesCanceledContext(t *testing.T) {
	dir := t.TempDir()
	writeJVM(t, dir, "src/main/java/com/acme/app/C.java", "com.acme.app")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := Prefixes(ctx, dir, javaExt)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, got)
}

func TestPrefixesCanceledContextReturnsPartialPrefixes(t *testing.T) {
	dir := t.TempDir()
	writeJVM(t, dir, "a.java", "com.acme.first")
	writeJVM(t, dir, "b.java", "com.acme.second")
	ctx := newCancelAfterContext(t.Context(), 3)
	t.Cleanup(ctx.cancel)

	got, err := Prefixes(ctx, dir, javaExt)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"com.acme.first"}, got)
}

func TestDeclaredPackageOfAnUnreadableFile(t *testing.T) {
	assert.Empty(t, declaredPackage(filepath.Join(t.TempDir(), "gone.java")))
}

func TestDeclaredPackageWithoutSemicolon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "K.kt")
	require.NoError(t, os.WriteFile(path, []byte("// header\npackage com.acme.k\n\nclass K\n"), 0o644))
	assert.Equal(t, "com.acme.k", declaredPackage(path))
}

func TestCommonPrefixes(t *testing.T) {
	tests := map[string]struct {
		in   []string
		want []string
	}{
		"empty":  {in: nil, want: nil},
		"single": {in: []string{"com.acme.billing.api"}, want: []string{"com.acme.billing.api"}},
		"branches under a shared prefix": {
			in:   []string{"com.acme.billing.api", "com.acme.billing.db", "com.acme.shared"},
			want: []string{"com.acme.billing", "com.acme.shared"},
		},
		"one name equals the shared prefix": {
			in:   []string{"com.acme", "com.acme.billing.api"},
			want: []string{"com.acme", "com.acme.billing"},
		},
		"nothing shared groups by first segment": {
			in:   []string{"com.acme.a", "com.acme.b", "org.other.x", "org.other.y.z"},
			want: []string{"com.acme.a", "com.acme.b", "org.other.x", "org.other.y"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, commonPrefixes(tt.in))
		})
	}
}
