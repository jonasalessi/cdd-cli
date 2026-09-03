package golang

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecID(t *testing.T) {
	assert.Equal(t, "go", string(Spec().ID))
	assert.False(t, Spec().IsApplicable("exception_handling"))
	assert.False(t, Spec().IsApplicable("inheritance"))
	assert.True(t, Spec().IsApplicable("code_branch"))
}

func TestDetectPackagesReadsTheModuleLine(t *testing.T) {
	got, err := Spec().DetectPackages(t.Context(), filepath.Join("testdata", "module"))
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/goonly"}, got)
}

func TestDetectPackagesWithoutAModuleLine(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.25.0\n"), 0o644))
	got, err := Spec().DetectPackages(t.Context(), dir)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDetectPackagesWithoutGoMod(t *testing.T) {
	got, err := Spec().DetectPackages(t.Context(), t.TempDir())
	require.NoError(t, err, "a missing go.mod is not an error")
	assert.Nil(t, got)
}

func TestDetectPackagesCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got, err := Spec().DetectPackages(ctx, filepath.Join("testdata", "module"))

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, got)
}
