package sum

import (
	"bufio"
	"context"
	"strings"

	"golang.org/x/xerrors"

	dependency "github.com/khulnasoft/dep-parser/pkg"
	ftypes "github.com/khulnasoft/dep-parser/pkg/types"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a go.sum file
func (p *Parser) Parse(_ context.Context, r xio.ReadSeekerAt) ([]ftypes.Package, []ftypes.Dependency, error) {
	var pkgs []ftypes.Package
	uniquePkgs := make(map[string]string)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		s := strings.Fields(line)
		if len(s) < 2 {
			continue
		}

		// go.sum records and sorts all non-major versions
		// with the latest version as last entry
		uniquePkgs[s[0]] = strings.TrimSuffix(s[1], "/go.mod")
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, xerrors.Errorf("scan error: %w", err)
	}

	for k, v := range uniquePkgs {
		pkgs = append(pkgs, ftypes.Package{
			ID:      dependency.ID(ftypes.GoModule, k, v),
			Name:    k,
			Version: v,
		})
	}

	return pkgs, nil, nil
}
