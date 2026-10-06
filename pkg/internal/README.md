# Vendored code

The packages in this directory are copied from upstream projects so that
`dep-parser` does not depend on them directly.

| Package | Upstream | Version | License |
| --- | --- | --- | --- |
| `pep440`, `pep440/part`, `pep440/prerelease` | `github.com/aquasecurity/go-pep440-version` | v0.0.1 | Apache-2.0 (`LICENSE-go-pep440-version`) |
| `semver`, `condaver`, `condaver/part`, `condaver/prerelease` | `github.com/aquasecurity/go-version` | v0.0.1 | Apache-2.0 (`LICENSE-go-version`) |

## What changed

Only import paths were rewritten to point at the local copies. No logic was
modified, so parsing behavior is identical to the upstream modules.

## Why these packages

Each is used for exactly one thing:

* `pep440` — `pkg/python/poetry` matches versions against PEP 440 specifiers.
* `semver` — `pkg/nodejs/pnpm` validates that a resolved version parses as semver.
* `condaver` — `pkg/conda/environment` validates that a dependency spec is a pinned version.

`semver` and `condaver` each carry their own copy of `part` and `prerelease`
because they are separate upstream modules. The two copies of `part` are
byte-identical; the copies of `prerelease` differ only in their import path.

## Updating

Re-copy from upstream, then rewrite the import paths:

```sh
GOMODCACHE=$(go env GOMODCACHE)
cp $GOMODCACHE/github.com/aquasecurity/go-version@v0.0.1/pkg/semver/*.go pkg/internal/semver/
cp $GOMODCACHE/github.com/aquasecurity/go-version@v0.0.1/pkg/version/*.go pkg/internal/condaver/
cp $GOMODCACHE/github.com/aquasecurity/go-version@v0.0.1/pkg/part/*.go pkg/internal/condaver/part/
cp $GOMODCACHE/github.com/aquasecurity/go-version@v0.0.1/pkg/prerelease/*.go pkg/internal/condaver/prerelease/
cp $GOMODCACHE/github.com/aquasecurity/go-pep440-version@v0.0.1/{version,specifier,specifier_option}.go pkg/internal/pep440/

sed -i '' 's|github.com/aquasecurity/go-version/pkg/part|github.com/khulnasoft/dep-parser/pkg/internal/condaver/part|g' pkg/internal/semver/*.go pkg/internal/condaver/*.go
sed -i '' 's|github.com/aquasecurity/go-version/pkg/prerelease|github.com/khulnasoft/dep-parser/pkg/internal/condaver/prerelease|g' pkg/internal/semver/*.go pkg/internal/condaver/*.go
sed -i '' 's|github.com/aquasecurity/go-version/pkg/part|github.com/khulnasoft/dep-parser/pkg/internal/pep440/part|g' pkg/internal/pep440/*.go
```

Then run `make test`. Coverage for `part` is 0% because upstream ships no tests
for it; it is exercised indirectly by the `pep440`, `semver`, and `condaver`
suites.