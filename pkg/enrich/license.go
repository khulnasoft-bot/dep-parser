package enrich

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/khulnasoft/dep-parser/pkg/types"
	xhttp "github.com/khulnasoft/dep-parser/pkg/x/http"
)

// Resolver fetches license metadata for a package from its ecosystem's
// public registry. Implementations must be safe for concurrent use.
type Resolver interface {
	// License returns the license expression/ID for pkg, or "" when unknown.
	License(ctx context.Context, pkg types.Package, lang types.LangType) (string, error)
}

// Registry resolves licenses by dispatching to per-ecosystem resolvers.
// Results are cached in-memory for the lifetime of the Registry.
type Registry struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]string
}

// NewRegistry returns a Registry using a default HTTP client. A nil client
// is replaced by a 10s-timeout client.
func NewRegistry(client *http.Client) *Registry {
	if client == nil {
		client = xhttp.Client(xhttp.WithTimeout(10 * time.Second))
	}
	return &Registry{client: client, cache: make(map[string]string)}
}

func (r *Registry) License(ctx context.Context, pkg types.Package, lang types.LangType) (string, error) {
	if len(pkg.Licenses) > 0 {
		return pkg.Licenses[0], nil
	}
	key := string(lang) + "|" + pkg.Name + "|" + pkg.Version
	r.mu.Lock()
	if v, ok := r.cache[key]; ok {
		r.mu.Unlock()
		return v, nil
	}
	r.mu.Unlock()

	var lic string
	var err error
	switch lang {
	case types.Npm, types.NodePkg, types.Pnpm, types.Yarn, types.Bun:
		lic, err = r.npmLicense(ctx, pkg)
	case types.Poetry, types.Pip, types.Pipenv, types.PyLock, types.Uv:
		lic, err = r.pypiLicense(ctx, pkg)
	case types.Pom, types.Gradle, types.Sbt, types.Jar:
		lic, err = r.mavenLicense(ctx, pkg)
	default:
		return "", nil
	}
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	r.cache[key] = lic
	r.mu.Unlock()
	return lic, nil
}

func (r *Registry) getJSON(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, rawURL)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (r *Registry) npmLicense(ctx context.Context, pkg types.Package) (string, error) {
	// Scoped names need the scope preserved: @scope/name -> @scope%2Fname
	u := "https://registry.npmjs.org/" + url.PathEscape(pkg.Name)
	if pkg.Version != "" {
		u += "/" + url.PathEscape(pkg.Version)
	}
	var doc struct {
		License any `json:"license"`
	}
	if err := r.getJSON(ctx, u, &doc); err != nil {
		return "", err
	}
	return licenseString(doc.License), nil
}

func (r *Registry) pypiLicense(ctx context.Context, pkg types.Package) (string, error) {
	u := "https://pypi.org/pypi/" + url.PathEscape(pkg.Name) + "/json"
	var doc struct {
		Info struct {
			License  string `json:"license"`
			Licenses []struct {
				License string `json:"license"`
			} `json:"licenses"`
		} `json:"info"`
	}
	if err := r.getJSON(ctx, u, &doc); err != nil {
		return "", err
	}
	if doc.Info.License != "" {
		return doc.Info.License, nil
	}
	for _, l := range doc.Info.Licenses {
		if l.License != "" {
			return l.License, nil
		}
	}
	return "", nil
}

func (r *Registry) mavenLicense(ctx context.Context, pkg types.Package) (string, error) {
	// pkg.Name may be "group:artifact"; resolve group/artifact first
	group, artifact := "", pkg.Name
	if i := strings.Index(pkg.Name, ":"); i > 0 {
		group, artifact = pkg.Name[:i], pkg.Name[i+1:]
	}
	q := "a:" + artifact
	if group != "" {
		q = "g:" + group + " AND " + q
	}
	u := "https://search.maven.org/solrsearch/select?q=" + url.QueryEscape(q) +
		"&rows=1&wt=json"
	var doc struct {
		Response struct {
			Docs []struct {
				GroupID       string `json:"g"`
				Name          string `json:"a"`
				LatestVersion string `json:"latestVersion"`
			} `json:"docs"`
		} `json:"response"`
	}
	if err := r.getJSON(ctx, u, &doc); err != nil || len(doc.Response.Docs) == 0 {
		return "", err
	}
	d := doc.Response.Docs[0]
	pomURL := fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.pom",
		strings.ReplaceAll(d.GroupID, ".", "/"), d.Name, d.LatestVersion, d.Name, d.LatestVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pomURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := r.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// POM licenses live at project/licenses/license/name; decode just enough XML.
	var pom struct {
		Licenses []struct {
			Name string `xml:"name"`
		} `xml:"licenses>license"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&pom); err != nil {
		return "", nil // unparseable POM is not fatal
	}
	var names []string
	for _, l := range pom.Licenses {
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	return strings.Join(names, " OR "), nil
}

// licenseString normalizes the heterogeneous npm "license" field (string or
// {"type": ...} object) into a single expression.
func licenseString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s, ok := t["type"].(string); ok {
			return s
		}
	case []any:
		var out []string
		for _, e := range t {
			if s := licenseString(e); s != "" {
				out = append(out, s)
			}
		}
		return strings.Join(out, " OR ")
	}
	return ""
}
