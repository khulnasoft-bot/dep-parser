package cargo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func TestParseDependencies(t *testing.T) {
	pkgs := map[string]cargoPkg{
		"serde":      {Name: "serde", Version: "1.0.100"},
		"unsafe-any": {Name: "unsafe-any", Version: "0.4.2"},
	}

	tests := []struct {
		name  string
		pkgID string
		pkg   cargoPkg
		want  *types.Dependency
	}{
		{
			name:  "unique dependency in new lock file",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"serde"}},
			want: &types.Dependency{
				ID:        "foo@1.0.0",
				DependsOn: []string{"serde@1.0.100"},
			},
		},
		{
			name:  "non-unique dependency in new lock file",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"unsafe-any 0.4.2"}},
			want: &types.Dependency{
				ID:        "foo@1.0.0",
				DependsOn: []string{"unsafe-any@0.4.2"},
			},
		},
		{
			name:  "old lock file with source",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"unsafe-any 0.4.2 (registry+https://github.com/rust-lang/crates.io-index)"}},
			want: &types.Dependency{
				ID:        "foo@1.0.0",
				DependsOn: []string{"unsafe-any@0.4.2"},
			},
		},
		{
			name:  "dependencies are sorted",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"serde", "unsafe-any 0.4.2"}},
			want: &types.Dependency{
				ID:        "foo@1.0.0",
				DependsOn: []string{"serde@1.0.100", "unsafe-any@0.4.2"},
			},
		},
		{
			name:  "no dependencies",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo"},
			want:  nil,
		},
		{
			name:  "unresolvable unique dependency is skipped",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"missing"}},
			want:  nil,
		},
		{
			name:  "malformed dependency is skipped",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"one two three four"}},
			want:  nil,
		},
		{
			name:  "all dependencies skipped",
			pkgID: "foo@1.0.0",
			pkg:   cargoPkg{Name: "foo", Dependencies: []string{"missing", "one two three four"}},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDependencies(tt.pkgID, tt.pkg, pkgs)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPropertyValue(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "quoted value", line: `name = "serde"`, want: "serde"},
		{name: "unquoted value", line: `version = 1.0.0`, want: "1.0.0"},
		{name: "indented", line: `  name = "serde"`, want: "serde"},
		{name: "too many separators", line: `name = "se=serde"`, want: ""},
		{name: "no separator", line: `name`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, propertyValue(tt.line))
		})
	}
}

func TestNaivePkgParserParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]pkgPosition
	}{
		{
			name: "single package",
			input: `[[package]]
name = "serde"
version = "1.0.100"
`,
			want: map[string]pkgPosition{
				"serde@1.0.100": {start: 1, end: 3},
			},
		},
		{
			name: "multiple packages",
			input: `[[package]]
name = "serde"
version = "1.0.100"

[[package]]
name = "unsafe-any"
version = "0.4.2"
`,
			want: map[string]pkgPosition{
				"serde@1.0.100":    {start: 1, end: 3},
				"unsafe-any@0.4.2": {start: 5, end: 7},
			},
		},
		{
			name: "package without trailing blank line",
			input: `[[package]]
name = "serde"
version = "1.0.100"
[[package]]
name = "unsafe-any"
version = "0.4.2"
`,
			want: map[string]pkgPosition{
				"serde@1.0.100":    {start: 1, end: 3},
				"unsafe-any@0.4.2": {start: 4, end: 6},
			},
		},
		{
			name: "table header without a name is ignored",
			input: `[[package]]
name = "serde"
version = "1.0.100"

[metadata]
`,
			want: map[string]pkgPosition{
				"serde@1.0.100": {start: 1, end: 3},
			},
		},
		{
			name:  "no packages",
			input: `# comment only`,
			want:  map[string]pkgPosition{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := naivePkgParser{r: strings.NewReader(tt.input)}
			got := parser.parse()
			require.Len(t, got, len(tt.want))
			for id, pos := range tt.want {
				assert.Equal(t, pos, got[id], id)
			}
		})
	}
}
