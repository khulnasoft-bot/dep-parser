package sbom

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

// spdxDoc is an SPDX 2.3 JSON document.
type spdxDoc struct {
	SPDXVersion       string             `json:"spdxVersion"`
	DataLicense       string             `json:"dataLicense"`
	SPDXID            string             `json:"SPDXID"`
	Name              string             `json:"name"`
	DocumentNamespace string             `json:"documentNamespace"`
	CreationInfo      spdxCreationInfo   `json:"creationInfo"`
	Packages          []spdxPackage      `json:"packages"`
	Relationships     []spdxRelationship `json:"relationships,omitempty"`
}

type spdxCreationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	SPDXID           string            `json:"SPDXID"`
	Name             string            `json:"name"`
	VersionInfo      string            `json:"versionInfo,omitempty"`
	DownloadLocation string            `json:"downloadLocation"`
	LicenseConcluded string            `json:"licenseConcluded"`
	LicenseDeclared  string            `json:"licenseDeclared"`
	CopyrightText    string            `json:"copyrightText"`
	ExternalRefs     []spdxExternalRef `json:"externalRefs,omitempty"`
}

type spdxExternalRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceType     string `json:"referenceType"`
	ReferenceLocator  string `json:"referenceLocator"`
}

type spdxRelationship struct {
	SPDXElementID        string `json:"spdxElementId"`
	RelationshipType     string `json:"relationshipType"`
	RelatedSPDXElementID string `json:"relatedSpdxElementId"`
}

var spdxIDInvalid = regexp.MustCompile(`[^a-zA-Z0-9.\-]`)

// spdxRef converts an arbitrary package ID into a valid SPDXID element.
func spdxRef(id string) string {
	id = spdxIDInvalid.ReplaceAllString(id, "-")
	if strings.Trim(id, "-") == "" {
		id = "unknown"
	}
	return "SPDXRef-Package-" + id
}

// MarshalSPDX encodes pkgs/deps as an SPDX 2.3 JSON document.
func MarshalSPDX(pkgs []types.Package, deps []types.Dependency, meta Metadata) ([]byte, error) {
	comps := normalize(pkgs, meta)
	name := meta.Name
	if name == "" {
		name = "dep-parser-sbom"
	}
	doc := spdxDoc{
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		SPDXID:            "SPDXRef-DOCUMENT",
		Name:              name,
		DocumentNamespace: "https://dep-parser.local/" + newUUID(),
		CreationInfo: spdxCreationInfo{
			Created:  time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			Creators: []string{"Tool: dep-parser-" + meta.ToolVersion},
		},
	}
	for _, c := range comps {
		p := spdxPackage{
			SPDXID:           spdxRef(c.id),
			Name:             c.name,
			VersionInfo:      c.version,
			DownloadLocation: "NOASSERTION",
			LicenseConcluded: "NOASSERTION",
			LicenseDeclared:  "NOASSERTION",
			CopyrightText:    "NOASSERTION",
		}
		if c.purl != "" {
			p.ExternalRefs = append(p.ExternalRefs, spdxExternalRef{
				ReferenceCategory: "PACKAGE-MANAGER",
				ReferenceType:     "purl",
				ReferenceLocator:  c.purl,
			})
		}
		if len(c.licenses) > 0 {
			p.LicenseDeclared = strings.Join(c.licenses, " OR ")
		}
		doc.Packages = append(doc.Packages, p)
	}
	for _, d := range deps {
		for _, target := range d.DependsOn {
			doc.Relationships = append(doc.Relationships, spdxRelationship{
				SPDXElementID:        spdxRef(d.ID),
				RelationshipType:     "DEPENDS_ON",
				RelatedSPDXElementID: spdxRef(target),
			})
		}
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal spdx: %w", err)
	}
	return out, nil
}
