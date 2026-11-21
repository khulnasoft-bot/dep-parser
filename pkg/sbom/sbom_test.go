package sbom

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func testPkgs() []types.Package {
	return []types.Package{
		{
			ID:        "lodash@4.17.21",
			Name:      "lodash",
			Version:   "4.17.21",
			Licenses:  []string{"MIT"},
			DependsOn: nil,
		},
		{
			ID:       "@scope/tool@1.0.0",
			Name:     "@scope/tool",
			Version:  "1.0.0",
			Dev:      true,
			FilePath: "node_modules/@scope/tool",
		},
		{
			ID:       "github.com/foo/bar@v1.2.3",
			Name:     "github.com/foo/bar",
			Version:  "v1.2.3",
			LangType: types.GoModule,
		},
	}
}

var testDeps = []types.Dependency{
	{ID: "app", DependsOn: []string{"lodash@4.17.21"}},
	{ID: "lodash@4.17.21", DependsOn: []string{"@scope/tool@1.0.0"}},
}

func TestMarshalCycloneDX(t *testing.T) {
	out, err := MarshalCycloneDX(testPkgs(), testDeps, Metadata{
		Name:        "my-app",
		Version:     "2.0.0",
		ToolVersion: "v0.1.0",
		Lang:        types.Npm,
	})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(out, &doc))
	assert.Equal(t, "CycloneDX", doc["bomFormat"])
	assert.Equal(t, "1.5", doc["specVersion"])

	components := doc["components"].([]any)
	require.Len(t, components, 3)
	first := components[0].(map[string]any)
	assert.Equal(t, "library", first["type"])
	assert.Equal(t, "lodash", first["name"])
	assert.Equal(t, "pkg:npm/lodash@4.17.21", first["purl"])
	assert.Equal(t, "MIT", first["licenses"].([]any)[0].(map[string]any)["license"].(map[string]any)["id"])

	// scoped npm name -> namespace purl
	second := components[1].(map[string]any)
	assert.Equal(t, "pkg:npm/@scope/tool@1.0.0", second["purl"])

	// per-package LangType overrides doc-level lang
	third := components[2].(map[string]any)
	assert.Equal(t, "github.com/foo/bar@v1.2.3", third["bom-ref"])
	assert.Equal(t, "pkg:golang/github.com/foo/bar@v1.2.3", third["purl"])

	deps := doc["dependencies"].([]any)
	require.Len(t, deps, 2)
	assert.Equal(t, "app", deps[0].(map[string]any)["ref"])
}

func TestMarshalCycloneDXGoLang(t *testing.T) {
	out, err := MarshalCycloneDX([]types.Package{{
		Name: "github.com/khulnasoft/go-version-lib", Version: "v1.0.0",
	}}, nil, Metadata{Lang: types.GoModule, ToolVersion: "test"})
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(out, &doc))
	comp := doc["components"].([]any)[0].(map[string]any)
	assert.Equal(t, "pkg:golang/github.com/khulnasoft/go-version-lib@v1.0.0", comp["purl"])
}

func TestMarshalSPDX(t *testing.T) {
	out, err := MarshalSPDX(testPkgs(), testDeps, Metadata{
		Name:        "my-app",
		ToolVersion: "v0.1.0",
		Lang:        types.Npm,
	})
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(out, &doc))
	assert.Equal(t, "SPDX-2.3", doc["spdxVersion"])
	assert.Equal(t, "CC0-1.0", doc["dataLicense"])
	assert.Equal(t, "my-app", doc["name"])

	pkgs := doc["packages"].([]any)
	require.Len(t, pkgs, 3)
	first := pkgs[0].(map[string]any)
	assert.Equal(t, "lodash", first["name"])
	assert.Equal(t, "MIT", first["licenseDeclared"])
	refs := first["externalRefs"].([]any)
	assert.Equal(t, "purl", refs[0].(map[string]any)["referenceType"])

	rels := doc["relationships"].([]any)
	require.Len(t, rels, 2)
	assert.Equal(t, "DEPENDS_ON", rels[0].(map[string]any)["relationshipType"])
}

func TestMarshalUnknownFormat(t *testing.T) {
	_, err := Marshal("bogus", nil, nil, Metadata{})
	require.Error(t, err)
}

func TestSPDXRefSanitizes(t *testing.T) {
	assert.Equal(t, "SPDXRef-Package-foo-bar-1.0", spdxRef("foo/bar:1.0"))
	assert.Equal(t, "SPDXRef-Package-unknown", spdxRef("!!!"))
	assert.Equal(t, "SPDXRef-Package-unknown", spdxRef(""))
}
