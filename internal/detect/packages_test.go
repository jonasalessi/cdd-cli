package detect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

func TestPackagesGoModule(t *testing.T) {
	got, err := Packages(fixture("go-only"), []config.Language{config.LangGo})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/goonly"}, got)
}

func TestPackagesJVMPrefixes(t *testing.T) {
	langs := []config.Language{config.LangJava, config.LangKotlin}
	got, err := Packages(fixture("java-kotlin"), langs)
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got)
}

func TestPackagesJavaOnlySkipsKotlinFiles(t *testing.T) {
	got, err := Packages(fixture("java-kotlin"), []config.Language{config.LangJava})
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme.billing", "com.acme.shared"}, got)
}

func TestPackagesTSConfigWithComments(t *testing.T) {
	got, err := Packages(fixture("ts"), []config.Language{config.LangTypeScript})
	require.NoError(t, err)
	assert.Equal(t, []string{"@app/", "@lib/"}, got)
}

func TestPackagesMissingFiles(t *testing.T) {
	got, err := Packages(fixture("empty"), []config.Language{
		config.LangGo, config.LangJava, config.LangKotlin, config.LangTypeScript,
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPackagesMixed(t *testing.T) {
	got, err := Packages(fixture("mixed"), []config.Language{config.LangGo, config.LangTypeScript})
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/mixed"}, got, "mixed has no tsconfig, only the module line")
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

func TestStripJSONC(t *testing.T) {
	in := "{\n// line comment\n\"a\": \"http://x//y\", /* block */\n\"b\": [1, 2,],\n}"
	assert.JSONEq(t, `{"a": "http://x//y", "b": [1, 2]}`, string(stripJSONC([]byte(in))))
}
