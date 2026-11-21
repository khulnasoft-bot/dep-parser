// Package parsers is the canonical registry of all dependency-file parsers.
// Both the CLI (cmd/dep-parser) and the recursive scanner (pkg/scan) use it.
package parsers

import (
	"context"
	"slices"

	"github.com/khulnasoft/dep-parser/pkg/parser/c/conan"
	"github.com/khulnasoft/dep-parser/pkg/parser/conda/environment"
	"github.com/khulnasoft/dep-parser/pkg/parser/conda/meta"
	"github.com/khulnasoft/dep-parser/pkg/parser/dart/pub"
	"github.com/khulnasoft/dep-parser/pkg/parser/dotnet/core_deps"
	"github.com/khulnasoft/dep-parser/pkg/parser/frameworks/wordpress"
	gobinary "github.com/khulnasoft/dep-parser/pkg/parser/golang/binary"
	"github.com/khulnasoft/dep-parser/pkg/parser/golang/mod"
	"github.com/khulnasoft/dep-parser/pkg/parser/golang/sum"
	gradlelock "github.com/khulnasoft/dep-parser/pkg/parser/gradle/lockfile"
	"github.com/khulnasoft/dep-parser/pkg/parser/hex/mix"
	"github.com/khulnasoft/dep-parser/pkg/parser/java/jar"
	"github.com/khulnasoft/dep-parser/pkg/parser/java/pom"
	juliamanifest "github.com/khulnasoft/dep-parser/pkg/parser/julia/manifest"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/bun"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/npm"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/packagejson"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/pnpm"
	"github.com/khulnasoft/dep-parser/pkg/parser/nodejs/yarn"
	"github.com/khulnasoft/dep-parser/pkg/parser/nuget/config"
	"github.com/khulnasoft/dep-parser/pkg/parser/nuget/lock"
	"github.com/khulnasoft/dep-parser/pkg/parser/nuget/packagesprops"
	"github.com/khulnasoft/dep-parser/pkg/parser/php/composer"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/packaging"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pip"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pipenv"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/poetry"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pylock"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/pyproject"
	"github.com/khulnasoft/dep-parser/pkg/parser/python/uv"
	"github.com/khulnasoft/dep-parser/pkg/parser/ruby/bundler"
	"github.com/khulnasoft/dep-parser/pkg/parser/ruby/gemspec"
	rustbinary "github.com/khulnasoft/dep-parser/pkg/parser/rust/binary"
	"github.com/khulnasoft/dep-parser/pkg/parser/rust/cargo"
	sbtlock "github.com/khulnasoft/dep-parser/pkg/parser/sbt/lockfile"
	"github.com/khulnasoft/dep-parser/pkg/parser/swift/cocoapods"
	"github.com/khulnasoft/dep-parser/pkg/parser/swift/swift"
	"github.com/khulnasoft/dep-parser/pkg/types"
	xio "github.com/khulnasoft/dep-parser/pkg/x/io"
)

// ParseFunc parses a dependency file. The returned values are
// ([]types.Package, []types.Dependency) for lockfile parsers; config-style
// parsers (package.json, pyproject.toml) return a single parsed document
// instead. Callers should type-assert before use.
type ParseFunc func(context.Context, xio.ReadSeekerAt) (any, any, error)

// Entry describes a registered parser.
type Entry struct {
	// Name is the CLI/registry identifier (e.g. "nodejs-npm").
	Name string
	// Lang is the ecosystem, used for purl/license enrichment. Empty for
	// config-style parsers that do not emit dependency packages.
	Lang types.LangType
	// Parse invokes the parser.
	Parse ParseFunc
}

func wrap(pkgs any, deps any, err error) (any, any, error) {
	return pkgs, deps, err
}

var entries = []Entry{
	{Name: "conan", Lang: types.Conan, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(conan.NewParser().Parse(ctx, r))
	}},
	{Name: "conda-env", Lang: types.Conda, Parse: func(_ context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, err := environment.NewParser().Parse(r)
		if err != nil {
			return nil, nil, err
		}
		return pkgs.Packages, nil, nil
	}},
	{Name: "conda-meta", Lang: types.Conda, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(meta.NewParser().Parse(ctx, r))
	}},
	{Name: "dart-pub", Lang: types.Pub, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(pub.NewParser(false).Parse(ctx, r))
	}},
	{Name: "dotnet", Lang: types.DotNetCore, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(core_deps.NewParser().Parse(ctx, r))
	}},
	{Name: "golang-binary", Lang: types.GoBinary, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(gobinary.NewParser().Parse(ctx, r))
	}},
	{Name: "golang-mod", Lang: types.GoModule, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, _, err := mod.NewParser(false, false).Parse(ctx, r)
		return pkgs, deps, err
	}},
	{Name: "golang-sum", Lang: types.GoModule, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(sum.NewParser().Parse(ctx, r))
	}},
	{Name: "gradle", Lang: types.Gradle, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(gradlelock.NewParser().Parse(ctx, r))
	}},
	{Name: "hex-mix", Lang: types.Hex, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(mix.NewParser().Parse(ctx, r))
	}},
	// java-jar runs offline: no Maven Central/Sonatype lookups.
	{Name: "java-jar", Lang: types.Jar, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(jar.NewParser(nil, jar.WithOffline(true)).Parse(ctx, r))
	}},
	{Name: "java-pom", Lang: types.Pom, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(pom.NewParser("").Parse(ctx, r))
	}},
	{Name: "julia", Lang: "", Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(juliamanifest.NewParser().Parse(ctx, r))
	}},
	{Name: "nodejs-bun", Lang: types.Bun, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(bun.NewParser().Parse(ctx, r))
	}},
	{Name: "nodejs-npm", Lang: types.Npm, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(npm.NewParser().Parse(ctx, r))
	}},
	{Name: "nodejs-pkg", Lang: "", Parse: func(_ context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkg, err := packagejson.NewParser().Parse(r)
		if err != nil {
			return nil, nil, err
		}
		return []any{pkg}, nil, nil
	}},
	{Name: "nodejs-pnpm", Lang: types.Pnpm, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(pnpm.NewParser().Parse(ctx, r))
	}},
	{Name: "nodejs-yarn", Lang: types.Yarn, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkgs, deps, _, err := yarn.NewParser().Parse(ctx, r)
		return pkgs, deps, err
	}},
	{Name: "nuget-config", Lang: types.NuGet, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(config.NewParser().Parse(ctx, r))
	}},
	{Name: "nuget-lock", Lang: types.NuGet, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(lock.NewParser().Parse(ctx, r))
	}},
	{Name: "nuget-props", Lang: types.NuGet, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(packagesprops.NewParser().Parse(ctx, r))
	}},
	{Name: "php-composer", Lang: types.Composer, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(composer.NewParser().Parse(ctx, r))
	}},
	{Name: "python-pkg", Lang: types.Pip, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(packaging.NewParser().Parse(ctx, r))
	}},
	{Name: "python-pip", Lang: types.Pip, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(pip.NewParser(false).Parse(ctx, r))
	}},
	{Name: "python-pipenv", Lang: types.Pipenv, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(pipenv.NewParser().Parse(ctx, r))
	}},
	{Name: "python-poetry", Lang: types.Poetry, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(poetry.NewParser().Parse(ctx, r))
	}},
	{Name: "python-pylock", Lang: types.PyLock, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(pylock.NewParser().Parse(ctx, r))
	}},
	{Name: "python-pyproject", Lang: "", Parse: func(_ context.Context, r xio.ReadSeekerAt) (any, any, error) {
		conf, err := pyproject.NewParser().Parse(r)
		if err != nil {
			return nil, nil, err
		}
		return []any{conf}, nil, nil
	}},
	{Name: "python-uv", Lang: types.Uv, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(uv.NewParser().Parse(ctx, r))
	}},
	{Name: "ruby-bundler", Lang: types.Bundler, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(bundler.NewParser().Parse(ctx, r))
	}},
	{Name: "ruby-gemspec", Lang: types.Bundler, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(gemspec.NewParser().Parse(ctx, r))
	}},
	{Name: "rust-binary", Lang: types.RustBinary, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(rustbinary.NewParser().Parse(ctx, r))
	}},
	{Name: "rust-cargo", Lang: types.Cargo, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(cargo.NewParser().Parse(ctx, r))
	}},
	{Name: "sbt", Lang: types.Sbt, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(sbtlock.NewParser().Parse(ctx, r))
	}},
	{Name: "swift", Lang: types.Swift, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(swift.NewParser().Parse(ctx, r))
	}},
	{Name: "swift-cocoapods", Lang: types.Cocoapods, Parse: func(ctx context.Context, r xio.ReadSeekerAt) (any, any, error) {
		return wrap(cocoapods.NewParser().Parse(ctx, r))
	}},
	{Name: "wordpress", Lang: "", Parse: func(_ context.Context, r xio.ReadSeekerAt) (any, any, error) {
		pkg, err := wordpress.Parse(r)
		if err != nil {
			return nil, nil, err
		}
		return []any{pkg}, nil, nil
	}},
}

// Registry maps parser name -> Entry.
var Registry = func() map[string]Entry {
	m := make(map[string]Entry, len(entries))
	for _, e := range entries {
		m[e.Name] = e
	}
	return m
}()

// Names returns all registered parser names, sorted.
func Names() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
