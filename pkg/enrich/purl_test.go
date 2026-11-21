package enrich

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestPURL(t *testing.T) {
	tests := []struct {
		name string
		pkg  types.Package
		lang types.LangType
		want string
	}{
		{"npm plain", types.Package{Name: "lodash", Version: "4.17.21"}, types.Npm, "pkg:npm/lodash@4.17.21"},
		{"npm scoped", types.Package{Name: "@scope/tool", Version: "1.0.0"}, types.Yarn, "pkg:npm/@scope/tool@1.0.0"},
		{"golang module", types.Package{Name: "github.com/foo/bar"}, types.GoModule, "pkg:golang/github.com/foo/bar"},
		{"maven coords", types.Package{Name: "org.apache.commons:commons-lang3", Version: "3.12.0"}, types.Pom, "pkg:maven/org.apache.commons/commons-lang3@3.12.0"},
		{"pypi", types.Package{Name: "requests", Version: "2.31.0"}, types.Pip, "pkg:pypi/requests@2.31.0"},
		{"cargo", types.Package{Name: "serde", Version: "1.0.0"}, types.Cargo, "pkg:cargo/serde@1.0.0"},
		{"composer", types.Package{Name: "vendor/pkg", Version: "1.0"}, types.Composer, "pkg:composer/vendor/pkg@1.0"},
		{"gem", types.Package{Name: "rails", Version: "7.0"}, types.Bundler, "pkg:gem/rails@7.0"},
		{"pkg LangType fallback", types.Package{Name: "lodash", Version: "1.0", LangType: types.Npm}, "", "pkg:npm/lodash@1.0"},
		{"unknown lang", types.Package{Name: "x", Version: "1"}, types.LangType("mystery"), ""},
		{"empty name", types.Package{Version: "1"}, types.Npm, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PURL(tt.pkg, tt.lang))
		})
	}
}

func TestLicenseString(t *testing.T) {
	assert.Equal(t, "MIT", licenseString("MIT"))
	assert.Equal(t, "Apache-2.0", licenseString(map[string]any{"type": "Apache-2.0"}))
	assert.Equal(t, "MIT OR Apache-2.0", licenseString([]any{"MIT", map[string]any{"type": "Apache-2.0"}}))
	assert.Equal(t, "", licenseString(nil))
	assert.Equal(t, "", licenseString(42))
}
