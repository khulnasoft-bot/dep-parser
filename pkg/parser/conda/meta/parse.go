package meta

import (
	"context"
	"encoding/json"

	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/licensing"
	"github.com/khulnasoft/dep-parser/pkg/types"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

type packageJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

// Parse parses Anaconda (a.k.a. conda) environment metadata.
// e.g. <conda-root>/envs/<env>/conda-meta/<package>.json
// For details see https://conda.io/projects/conda/en/latest/user-guide/concepts/environments.html
func (p *Parser) Parse(_ context.Context, r xio.ReadSeekerAt) ([]types.Package, []types.Dependency, error) {
	var data packageJSON
	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, nil, xerrors.Errorf("JSON decode error: %w", err)
	}

	if data.Name == "" || data.Version == "" {
		return nil, nil, xerrors.Errorf("unable to parse conda package")
	}

	return []types.Package{
		{
			Name:     data.Name,
			Version:  data.Version,
			LangType: types.Conda,
			Licenses: licensing.SplitLicenses(data.License),
		},
	}, nil, nil
}
