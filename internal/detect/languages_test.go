package detect

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

func fixture(name string) string {
	return filepath.Join("testdata", name)
}

func TestLanguagesPerFixture(t *testing.T) {
	tests := map[string]struct {
		root string
		want map[config.Language]int
	}{
		"go only, node_modules skipped": {
			root: "go-only",
			want: map[config.Language]int{config.LangGo: 3},
		},
		"java and kotlin, kts counted": {
			root: "java-kotlin",
			want: map[config.Language]int{config.LangJava: 3, config.LangKotlin: 2},
		},
		"typescript variants": {
			root: "ts",
			want: map[config.Language]int{config.LangTypeScript: 4},
		},
		"mixed tree": {
			root: "mixed",
			want: map[config.Language]int{config.LangGo: 1, config.LangTypeScript: 1},
		},
		"empty tree": {
			root: "empty",
			want: map[config.Language]int{},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			d, err := Languages(context.Background(), fixture(tt.root))
			require.NoError(t, err)
			assert.Equal(t, tt.want, d.Counts)
			assert.False(t, d.Truncated)
			assert.GreaterOrEqual(t, d.Elapsed, time.Duration(0))
		})
	}
}

func TestLanguagesOrder(t *testing.T) {
	d := Detected{Counts: map[config.Language]int{
		config.LangTypeScript: 1,
		config.LangGo:         2,
	}}
	assert.Equal(t, []config.Language{config.LangGo, config.LangTypeScript}, d.Languages())
	assert.Empty(t, Detected{}.Languages())
}

func TestLanguagesExpiredContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, err := Languages(ctx, fixture("go-only"))
	require.NoError(t, err, "a cut-short scan is not a failure")
	assert.True(t, d.Truncated)
	assert.Empty(t, d.Counts)
}

func TestLanguagesDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	d, err := Languages(ctx, fixture("mixed"))
	require.NoError(t, err)
	assert.True(t, d.Truncated)
}

func TestLanguagesMissingRoot(t *testing.T) {
	_, err := Languages(context.Background(), fixture("does-not-exist"))
	assert.Error(t, err)
}
