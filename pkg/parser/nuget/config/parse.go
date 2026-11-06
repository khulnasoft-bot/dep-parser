package config

import (
	"context"
	"encoding/xml"

	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/parser/utils"
	"github.com/khulnasoft/dep-parser/pkg/types"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

type cfgPackageReference struct {
	XMLName         xml.Name `xml:"package"`
	TargetFramework string   `xml:"targetFramework,attr"`
	Version         string   `xml:"version,attr"`
	DevDependency   bool     `xml:"developmentDependency,attr"`
	ID              string   `xml:"id,attr"`
}

type config struct {
	XMLName  xml.Name              `xml:"packages"`
	Packages []cfgPackageReference `xml:"package"`
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(_ context.Context, r xio.ReadSeekerAt) ([]types.Package, []types.Dependency, error) {
	var cfgData config
	if err := xml.NewDecoder(r).Decode(&cfgData); err != nil {
		return nil, nil, xerrors.Errorf("failed to decode .config file: %w", err)
	}

	var pkgs []types.Package
	for _, pkg := range cfgData.Packages {
		if pkg.ID == "" || pkg.DevDependency {
			continue
		}

		pkgs = append(pkgs, types.Package{
			Name:    pkg.ID,
			Version: pkg.Version,
		})
	}

	return utils.UniquePackages(pkgs), nil, nil
}
