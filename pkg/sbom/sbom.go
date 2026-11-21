// Package sbom converts parser output into Software Bill of Materials
// documents (CycloneDX and SPDX).
package sbom

import (
	"fmt"

	"github.com/khulnasoft/dep-parser/pkg/enrich"
	"github.com/khulnasoft/dep-parser/pkg/types"
)

// Format identifies an SBOM output format.
type Format string

const (
	FormatCycloneDX Format = "cyclonedx"
	FormatSPDX      Format = "spdx"
)

// Marshal encodes pkgs/deps in the requested format.
func Marshal(format Format, pkgs []types.Package, deps []types.Dependency, meta Metadata) ([]byte, error) {
	switch format {
	case FormatCycloneDX:
		return MarshalCycloneDX(pkgs, deps, meta)
	case FormatSPDX:
		return MarshalSPDX(pkgs, deps, meta)
	default:
		return nil, fmt.Errorf("unknown sbom format: %q", format)
	}
}

// Metadata describes the document being generated.
type Metadata struct {
	// Name is the document/component name (e.g. the scanned project).
	Name string
	// Version is the document version, if known.
	Version string
	// ToolVersion is the dep-parser version, reported in tools/creators.
	ToolVersion string
	// Lang is the ecosystem the packages came from; used for purl generation.
	Lang types.LangType
}

// component is the internal normalized view used by both emitters.
type component struct {
	id       string // bom-ref / SPDXID / dependency ref
	name     string
	version  string
	purl     string
	licenses []string
	dev      bool
	filePath string
}

func normalize(pkgs []types.Package, meta Metadata) []component {
	out := make([]component, 0, len(pkgs))
	for _, p := range pkgs {
		lang := meta.Lang
		if p.LangType != "" {
			lang = p.LangType
		}
		c := component{
			id:       p.ID,
			name:     p.Name,
			version:  p.Version,
			purl:     enrich.PURL(p, lang),
			licenses: p.Licenses,
			dev:      p.Dev,
			filePath: p.FilePath,
		}
		if c.id == "" {
			c.id = c.purl
		}
		if c.id == "" {
			c.id = c.name + "@" + c.version
		}
		out = append(out, c)
	}
	return out
}
