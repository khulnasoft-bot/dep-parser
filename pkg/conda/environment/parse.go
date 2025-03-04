package environment

import (
	"sort"
	"strings"

	"github.com/aquasecurity/go-version/pkg/version"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	dio "github.com/khulnasoft/dep-parser/pkg/io"
	"github.com/khulnasoft/dep-parser/pkg/types"
)

type environment struct {
	Entries []Entry `yaml:"dependencies"`
	Prefix  string  `yaml:"prefix"`
}

type Entry struct {
	Dependencies []Dependency
}

type Dependency struct {
	Value string
	Line  int
}

type Packages struct {
	Packages []types.Library
	Prefix   string
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(r dio.ReadSeekerAt) (Packages, error) {
	var env environment
	if err := yaml.NewDecoder(r).Decode(&env); err != nil {
		return Packages{}, xerrors.Errorf("unable to decode conda environment.yml file: %w", err)
	}

	var pkgs []types.Library
	for _, entry := range env.Entries {
		for _, dep := range entry.Dependencies {
			pkg := p.toPackage(dep)
			// Skip empty pkgs
			if pkg.Name == "" {
				continue
			}
			pkgs = append(pkgs, pkg)
		}
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].Name < pkgs[j].Name
	})
	return Packages{
		Packages: pkgs,
		Prefix:   env.Prefix,
	}, nil
}

func (p *Parser) toPackage(dep Dependency) types.Library {
	name, ver := p.parseDependency(dep.Value)
	if ver == "" {
		// Remove logger warning as it's not needed
	}
	return types.Library{
		Name:    name,
		Version: ver,
		Locations: types.Locations{
			{
				StartLine: dep.Line,
				EndLine:   dep.Line,
			},
		},
	}
}

// parseDependency parses the dependency line and returns the name and the pinned version.
// The version range is not supported. It parses only the pinned version.
// e.g.
//   - numpy 1.8.1
//   - numpy ==1.8.1
//   - numpy 1.8.1 py27_0
//   - numpy=1.8.1=py27_0
//
// cf. https://docs.conda.io/projects/conda-build/en/latest/resources/package-spec.html#examples-of-package-specs
func (*Parser) parseDependency(line string) (string, string) {
	line = strings.NewReplacer(">", " >", "<", " <", "=", " ").Replace(line)
	parts := strings.Fields(line)
	name := parts[0]
	if len(parts) == 1 {
		return name, ""
	}
	if _, err := version.Parse(parts[1]); err != nil {
		return name, ""
	}
	return name, parts[1]
}

func (e *Entry) UnmarshalYAML(node *yaml.Node) error {
	var dependencies []Dependency
	// cf. https://github.com/go-yaml/yaml/blob/f6f7691b1fdeb513f56608cd2c32c51f8194bf51/resolve.go#L70-L81
	switch node.Tag {
	case "!!str":
		dependencies = append(dependencies, Dependency{
			Value: node.Value,
			Line:  node.Line,
		})
	case "!!map":
		if node.Content != nil {
			// Map key is package manager (e.g. pip). So we need to store only map values (dependencies).
			// e.g. dependencies:
			//  	  - pip:
			//     	    - pandas==2.1.4
			if node.Content[1].Tag != "!!seq" { // Conda supports only map[string][]string format.
				return xerrors.Errorf("unsupported dependency type %q on line %d", node.Content[1].Tag, node.Content[1].Line)
			}

			for _, depContent := range node.Content[1].Content {
				if depContent.Tag != "!!str" {
					return xerrors.Errorf("unsupported dependency type %q on line %d", depContent.Tag, depContent.Line)
				}

				dependencies = append(dependencies, Dependency{
					Value: depContent.Value,
					Line:  depContent.Line,
				})
			}
		}
	default:
		return xerrors.Errorf("unsupported dependency type %q on line %d", node.Tag, node.Line)
	}

	e.Dependencies = dependencies
	return nil
}
