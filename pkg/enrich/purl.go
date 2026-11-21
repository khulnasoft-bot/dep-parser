// Package enrich derives additional package metadata (purl, licenses)
// that parsers do not emit themselves.
package enrich

import (
	"strings"

	"github.com/khulnasoft/dep-parser/pkg/types"
	"github.com/khulnasoft/go-version-lib/pkg/purl"
)

// PURL returns a Package URL for pkg in the given ecosystem. lang may be
// empty, in which case pkg.LangType is used. Returns "" when the language
// type has no well-known purl mapping or required fields are missing.
func PURL(pkg types.Package, lang types.LangType) string {
	if lang == "" {
		lang = pkg.LangType
	}
	p := purl.PackageURL{
		Type:    purlType(lang),
		Name:    pkg.Name,
		Version: pkg.Version,
	}
	if p.Type == "" || p.Name == "" {
		return ""
	}
	p.Namespace, p.Name = splitNamespace(lang, pkg.Name)
	return p.String()
}

func purlType(l types.LangType) string {
	switch l {
	case types.Npm, types.NodePkg, types.Pnpm, types.Yarn, types.Bun:
		return "npm"
	case types.GoModule, types.GoBinary:
		return "golang"
	case types.Cargo, types.RustBinary:
		return "cargo"
	case types.Pub:
		return "pub"
	case types.Hex:
		return "hex"
	case types.Bundler:
		return "gem"
	case types.Composer:
		return "composer"
	case types.Poetry, types.Pip, types.Pipenv, types.PyLock, types.Uv:
		return "pypi"
	case types.NuGet, types.DotNetCore:
		return "nuget"
	case types.Pom, types.Gradle, types.Sbt, types.Jar:
		return "maven"
	case types.Conan:
		return "conan"
	case types.Conda:
		return "conda"
	case types.Cocoapods:
		return "cocoapods"
	case types.Swift:
		return "swift"
	default:
		return ""
	}
}

// splitNamespace extracts the purl namespace from names that encode it:
//   - npm scoped packages: "@scope/name" -> namespace "@scope"
//   - maven coordinates: "group:artifact" -> namespace group
//   - golang modules: "github.com/foo/bar" -> namespace "github.com/foo"
//   - composer packages: "vendor/pkg" -> namespace vendor
func splitNamespace(l types.LangType, name string) (namespace, pkgName string) {
	switch l {
	case types.Npm, types.NodePkg, types.Pnpm, types.Yarn, types.Bun:
		if strings.HasPrefix(name, "@") {
			if i := strings.Index(name[1:], "/"); i >= 0 {
				return name[:i+1], name[i+2:]
			}
		}
	case types.Pom, types.Gradle, types.Sbt, types.Jar:
		if i := strings.Index(name, ":"); i > 0 {
			return name[:i], name[i+1:]
		}
	case types.GoModule, types.GoBinary:
		if i := strings.LastIndex(name, "/"); i > 0 {
			return name[:i], name[i+1:]
		}
	case types.Composer:
		if i := strings.Index(name, "/"); i > 0 {
			return name[:i], name[i+1:]
		}
	}
	return "", name
}
