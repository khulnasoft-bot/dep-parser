package sum

import (
	"bufio"
	"sort"
	"strings"

	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/golang/mod"
	dio "github.com/khulnasoft/dep-parser/pkg/io"
	"github.com/khulnasoft/dep-parser/pkg/types"
)

type Parser struct{}

func NewParser() types.Parser {
	return &Parser{}
}

// Parse parses a go.sum file
func (p *Parser) Parse(r dio.ReadSeekerAt) ([]types.Library, []types.Dependency, error) {
	var libs []types.Library
	uniqueLibs := make(map[string]string)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		s := strings.Fields(line)
		if len(s) < 2 {
			continue
		}

		// go.sum records and sorts all non-major versions
		// with the latest version as last entry
		uniqueLibs[s[0]] = strings.TrimSuffix(strings.TrimPrefix(s[1], "v"), "/go.mod")
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, xerrors.Errorf("scan error: %w", err)
	}

	for k, v := range uniqueLibs {
		libs = append(libs, types.Library{
			ID:      mod.ModuleID(k, v),
			Name:    k,
			Version: v,
		})
	}

	// uniqueLibs is a map, so its iteration order is randomized. Sort to keep
	// the output stable across runs.
	sort.Sort(types.Libraries(libs))

	return libs, nil, nil
}
