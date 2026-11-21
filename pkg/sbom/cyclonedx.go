package sbom

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	s := hex.EncodeToString(b[:])
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}

// cycloneDXDoc is a CycloneDX 1.5 JSON document.
type cycloneDXDoc struct {
	BOMFormat    string                `json:"bomFormat"`
	SpecVersion  string                `json:"specVersion"`
	SerialNumber string                `json:"serialNumber"`
	Version      int                   `json:"version"`
	Metadata     cycloneDXMetadata     `json:"metadata"`
	Components   []cycloneDXComponent  `json:"components"`
	Dependencies []cycloneDXDependency `json:"dependencies,omitempty"`
}

type cycloneDXMetadata struct {
	Timestamp string              `json:"timestamp"`
	Tools     cycloneDXTool       `json:"tools"`
	Component *cycloneDXComponent `json:"component,omitempty"`
}

type cycloneDXTool struct {
	Components []cycloneDXComponent `json:"components"`
}

type cycloneDXComponent struct {
	Type       string              `json:"type"`
	BOMRef     string              `json:"bom-ref"`
	Name       string              `json:"name"`
	Version    string              `json:"version,omitempty"`
	PURL       string              `json:"purl,omitempty"`
	Licenses   []cycloneDXLicense  `json:"licenses,omitempty"`
	Properties []cycloneDXProperty `json:"properties,omitempty"`
}

type cycloneDXLicense struct {
	License cycloneDXLicenseID `json:"license"`
}

type cycloneDXLicenseID struct {
	ID string `json:"id"`
}

type cycloneDXProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cycloneDXDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn,omitempty"`
}

// MarshalCycloneDX encodes pkgs/deps as a CycloneDX 1.5 JSON document.
func MarshalCycloneDX(pkgs []types.Package, deps []types.Dependency, meta Metadata) ([]byte, error) {
	comps := normalize(pkgs, meta)
	doc := cycloneDXDoc{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: "urn:uuid:" + newUUID(),
		Version:      1,
		Metadata: cycloneDXMetadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Tools: cycloneDXTool{Components: []cycloneDXComponent{{
				Type:    "application",
				BOMRef:  "pkg:generic/dep-parser@" + meta.ToolVersion,
				Name:    "dep-parser",
				Version: meta.ToolVersion,
			}}},
		},
	}
	if meta.Name != "" {
		doc.Metadata.Component = &cycloneDXComponent{
			Type:    "application",
			BOMRef:  "pkg:generic/" + meta.Name,
			Name:    meta.Name,
			Version: meta.Version,
		}
	}
	for _, c := range comps {
		cc := cycloneDXComponent{
			Type:    "library",
			BOMRef:  c.id,
			Name:    c.name,
			Version: c.version,
			PURL:    c.purl,
		}
		for _, l := range c.licenses {
			cc.Licenses = append(cc.Licenses, cycloneDXLicense{License: cycloneDXLicenseID{ID: l}})
		}
		if c.dev {
			cc.Properties = append(cc.Properties, cycloneDXProperty{Name: "dep-parser:dev", Value: "true"})
		}
		if c.filePath != "" {
			cc.Properties = append(cc.Properties, cycloneDXProperty{Name: "dep-parser:file-path", Value: c.filePath})
		}
		doc.Components = append(doc.Components, cc)
	}
	for _, d := range deps {
		doc.Dependencies = append(doc.Dependencies, cycloneDXDependency{
			Ref:       d.ID,
			DependsOn: d.DependsOn,
		})
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal cyclonedx: %w", err)
	}
	return out, nil
}
