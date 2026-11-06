package lockfile

import (
	"bufio"
	"context"
	"strings"

	dependency "github.com/khulnasoft/dep-parser/pkg"
	"github.com/khulnasoft/dep-parser/pkg/parser/utils"
	ftypes "github.com/khulnasoft/dep-parser/pkg/types"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (Parser) Parse(_ context.Context, r xio.ReadSeekerAt) ([]ftypes.Package, []ftypes.Dependency, error) {
	var pkgs []ftypes.Package
	scanner := bufio.NewScanner(r)
	var lineNum int
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") { // skip comments
			continue
		}

		// dependency format: group:artifact:version=classPaths
		dep := strings.Split(line, ":")
		if len(dep) != 3 { // skip the last line with lists of empty configurations
			continue
		}

		name := strings.Join(dep[:2], ":")
		version, classPathsString, _ := strings.Cut(dep[2], "=")

		dev := true

		for classPath := range strings.SplitSeq(classPathsString, ",") {
			if !strings.HasPrefix(classPath, "test") {
				dev = false
				break
			}
		}

		pkgs = append(pkgs, ftypes.Package{
			ID:      dependency.ID(ftypes.Gradle, name, version),
			Name:    name,
			Version: version,
			Dev:     dev,
			Locations: []ftypes.Location{
				{
					StartLine: lineNum,
					EndLine:   lineNum,
				},
			},
			Relationship: ftypes.RelationshipUnknown,
		})

	}
	return utils.UniquePackages(pkgs), nil, nil
}
