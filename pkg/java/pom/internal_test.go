package pom

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateVariable(t *testing.T) {
	t.Run("no variables", func(t *testing.T) {
		assert.Equal(t, "1.0.0", evaluateVariable("1.0.0", nil, nil))
	})

	t.Run("known property", func(t *testing.T) {
		props := map[string]string{"ver": "1.0.0"}
		assert.Equal(t, "1.0.0", evaluateVariable("${ver}", props, nil))
	})

	t.Run("unknown property resolves to empty", func(t *testing.T) {
		assert.Equal(t, "", evaluateVariable("${missing}", map[string]string{}, nil))
	})

	t.Run("nested property", func(t *testing.T) {
		props := map[string]string{"a": "${b}", "b": "final"}
		assert.Equal(t, "final", evaluateVariable("${a}", props, nil))
	})

	t.Run("environment variable", func(t *testing.T) {
		t.Setenv("DEP_PARSER_TEST_VAR", "from-env")
		assert.Equal(t, "from-env", evaluateVariable("${env.DEP_PARSER_TEST_VAR}", nil, nil))
	})

	t.Run("looped property resolves to empty", func(t *testing.T) {
		props := map[string]string{"a": "${b}", "b": "${a}"}
		assert.Equal(t, "", evaluateVariable("${a}", props, nil))
	})
}

func TestArtifactHelpers(t *testing.T) {
	t.Run("IsEmpty", func(t *testing.T) {
		assert.True(t, artifact{}.IsEmpty())
		assert.True(t, artifact{GroupID: "g", ArtifactID: "a"}.IsEmpty())
		assert.True(t, artifact{GroupID: "g", Version: newVersion("1")}.IsEmpty())
		assert.True(t, artifact{ArtifactID: "a", Version: newVersion("1")}.IsEmpty())
		assert.False(t, artifact{GroupID: "g", ArtifactID: "a", Version: newVersion("1")}.IsEmpty())
	})

	t.Run("Name and String", func(t *testing.T) {
		a := artifact{GroupID: "org.example", ArtifactID: "lib", Version: newVersion("1.0.0")}
		assert.Equal(t, "org.example:lib", a.Name())
		assert.Equal(t, "org.example:lib:1.0.0", a.String())
	})

	t.Run("JoinLicenses", func(t *testing.T) {
		assert.Equal(t, "MIT, Apache-2.0", artifact{Licenses: []string{"MIT", "Apache-2.0"}}.JoinLicenses())
		assert.Equal(t, "", artifact{}.JoinLicenses())
	})

	t.Run("ToPOMLicenses", func(t *testing.T) {
		got := artifact{Licenses: []string{"MIT", "Apache-2.0"}}.ToPOMLicenses()
		assert.Equal(t, pomLicenses{
			License: []pomLicense{{Name: "MIT"}, {Name: "Apache-2.0"}},
		}, got)
	})

	t.Run("Inherit fills the gaps", func(t *testing.T) {
		parent := artifact{
			GroupID:  "org.parent",
			Licenses: []string{"MIT"},
			Version:  newVersion("2.0.0"),
		}

		child := artifact{ArtifactID: "lib"}
		got := child.Inherit(parent)
		assert.Equal(t, "org.parent", got.GroupID)
		assert.Equal(t, "lib", got.ArtifactID)
		assert.Equal(t, "2.0.0", got.Version.String())
		assert.Equal(t, []string{"MIT"}, got.Licenses)
	})

	t.Run("Inherit keeps the child's own values", func(t *testing.T) {
		parent := artifact{
			GroupID:  "org.parent",
			Licenses: []string{"MIT"},
			Version:  newVersion("2.0.0"),
		}

		child := artifact{
			GroupID:  "org.child",
			Licenses: []string{"Apache-2.0"},
			Version:  newVersion("1.0.0"),
		}
		got := child.Inherit(parent)
		assert.Equal(t, "org.child", got.GroupID)
		assert.Equal(t, []string{"Apache-2.0"}, got.Licenses)
		assert.Equal(t, "1.0.0", got.Version.String())
	})
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantHard bool
	}{
		{name: "plain version", input: "1.0.0", want: "1.0.0"},
		{name: "hard requirement", input: "[1.0.0]", want: "1.0.0", wantHard: true},
		{name: "unsupported range is dropped", input: "1.0.0,2.0.0", want: ""},
		{name: "parenthesised range is dropped", input: "(1.0.0)", want: ""},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newVersion(tt.input)
			assert.Equal(t, tt.want, got.String())
			assert.Equal(t, tt.wantHard, got.hard)
		})
	}
}

func TestVersionShouldOverride(t *testing.T) {
	assert.True(t, newVersion("1.0.0").shouldOverride(newVersion("[2.0.0]")))
	assert.False(t, newVersion("[1.0.0]").shouldOverride(newVersion("2.0.0")))
	assert.False(t, newVersion("1.0.0").shouldOverride(newVersion("2.0.0")))
}

func TestNewArtifactEvaluatesProperties(t *testing.T) {
	props := map[string]string{
		"group":    "org.example",
		"artifact": "lib",
		"version":  "1.0.0",
	}

	got := newArtifact("${group}", "${artifact}", "${version}", []string{"MIT"}, props)
	assert.Equal(t, "org.example", got.GroupID)
	assert.Equal(t, "lib", got.ArtifactID)
	assert.Equal(t, "1.0.0", got.Version.String())
}

func TestPomDependencyResolve(t *testing.T) {
	dep := pomDependency{
		GroupID:    "org.example",
		ArtifactID: "lib",
		Version:    "${lib.version}",
		Scope:      "compile",
	}

	t.Run("variables are evaluated", func(t *testing.T) {
		got := dep.Resolve(map[string]string{"lib.version": "1.0.0"}, nil, nil)
		assert.Equal(t, "1.0.0", got.Version)
		assert.Equal(t, "org.example:lib", got.Name())
	})

	t.Run("root dependencyManagement wins", func(t *testing.T) {
		rootDepManagement := []pomDependency{
			{
				GroupID:    "org.example",
				ArtifactID: "lib",
				Version:    "3.0.0",
				Scope:      "runtime",
				Optional:   true,
				Exclusions: pomExclusions{
					Exclusion: []pomExclusion{{GroupID: "org.slope", ArtifactID: "junit"}},
				},
			},
		}

		got := dep.Resolve(map[string]string{"lib.version": "1.0.0"}, nil, rootDepManagement)
		assert.Equal(t, "3.0.0", got.Version)
		assert.Equal(t, "runtime", got.Scope)
		assert.True(t, got.Optional)
		assert.Equal(t, pomExclusions{
			Exclusion: []pomExclusion{{GroupID: "org.slope", ArtifactID: "junit"}},
		}, got.Exclusions)
	})

	t.Run("inherits from the parent dependencyManagement", func(t *testing.T) {
		depManagement := []pomDependency{
			{
				GroupID:    "org.example",
				ArtifactID: "lib",
				Version:    "2.0.0",
				Scope:      "provided",
				Exclusions: pomExclusions{
					Exclusion: []pomExclusion{{GroupID: "org.slope", ArtifactID: "junit"}},
				},
			},
		}

		empty := pomDependency{GroupID: "org.example", ArtifactID: "lib"}
		got := empty.Resolve(nil, depManagement, nil)
		assert.Equal(t, "2.0.0", got.Version)
		assert.Equal(t, "provided", got.Scope)
		assert.Equal(t, pomExclusions{
			Exclusion: []pomExclusion{{GroupID: "org.slope", ArtifactID: "junit"}},
		}, got.Exclusions)

		// explicitly set fields are not overwritten by the managed dependency
		explicit := empty
		explicit.Version = "1.0.0"
		explicit.Scope = "compile"
		explicit.Exclusions = pomExclusions{
			Exclusion: []pomExclusion{{GroupID: "org.other", ArtifactID: "thing"}},
		}
		got = explicit.Resolve(nil, depManagement, nil)
		assert.Equal(t, "1.0.0", got.Version)
		assert.Equal(t, "compile", got.Scope)
		assert.Equal(t, pomExclusions{
			Exclusion: []pomExclusion{{GroupID: "org.other", ArtifactID: "thing"}},
		}, got.Exclusions)
	})
}

func TestFindDep(t *testing.T) {
	depManagement := []pomDependency{
		{GroupID: "org.example", ArtifactID: "lib", Version: "1.0.0"},
	}

	got, found := findDep("org.example:lib", depManagement)
	assert.True(t, found)
	assert.Equal(t, "1.0.0", got.Version)

	_, found = findDep("org.example:missing", depManagement)
	assert.False(t, found)
}

func TestPomPropertiesUnmarshalXMLError(t *testing.T) {
	decoder := xml.NewDecoder(strings.NewReader(`<properties><key>`))

	var props properties
	err := props.UnmarshalXML(decoder, xml.StartElement{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "XML decode error")
}

func TestPomDependenciesUnmarshalXMLError(t *testing.T) {
	decoder := xml.NewDecoder(strings.NewReader(`<dependencies><dependency><groupId>g</groupId></wrong></dependencies>`))

	var deps pomDependencies
	err := deps.UnmarshalXML(decoder, xml.StartElement{})
	require.Error(t, err)
	// note: the wrapped error loses its cause because the format verb has no
	// argument, so only the prefix is asserted here
	assert.ErrorContains(t, err, "Error decoding dependency")
}

func TestParsePomError(t *testing.T) {
	_, err := parsePom(strings.NewReader("this is not xml"))
	require.Error(t, err)
	assert.ErrorContains(t, err, "xml decode error")
}

func TestPomRepositoriesAndLicenses(t *testing.T) {
	content, err := parsePom(strings.NewReader(`<project>
  <artifactId>lib</artifactId>
  <version>1.0.0</version>
  <licenses>
    <license><name>MIT</name></license>
  </licenses>
  <repositories>
    <repository>
      <id>central</id>
      <url>https://repo.maven.apache.org/maven2</url>
    </repository>
    <repository>
      <id>internal</id>
      <url>https://internal.example/repo</url>
    </repository>
  </repositories>
</project>`))
	require.NoError(t, err)

	p := pom{content: content}
	assert.Equal(t, []string{"MIT"}, p.licenses())
	assert.Equal(t, []string{
		"https://repo.maven.apache.org/maven2",
		"https://internal.example/repo",
	}, p.repositories())
	assert.Equal(t, "lib", p.artifact().ArtifactID)
	assert.Equal(t, "1.0.0", p.artifact().Version.String())
}

func TestPomProjectProperties(t *testing.T) {
	content, err := parsePom(strings.NewReader(`<project>
  <groupId>org.example</groupId>
  <artifactId>lib</artifactId>
  <version>1.0.0</version>
  <properties>
    <custom>value</custom>
  </properties>
</project>`))
	require.NoError(t, err)

	p := pom{content: content}

	// groupId and version are exposed to properties, and project.* prefixed
	// keys are filtered out
	props := p.properties()
	assert.Equal(t, "value", props["custom"])
	assert.Equal(t, "org.example", props["groupId"])
	assert.Equal(t, "1.0.0", props["version"])
	// every key is also exposed with the project. prefix, e.g. ${project.version}
	assert.Equal(t, "1.0.0", props["project.version"])
	assert.Equal(t, "lib", props["project.artifactId"])
	assert.Equal(t, "value", props["project.custom"])
}

func TestPomInheritPropertyVersion(t *testing.T) {
	content, err := parsePom(strings.NewReader(`<project>
  <groupId>org.example</groupId>
  <artifactId>lib</artifactId>
  <version>${lib.version}</version>
</project>`))
	require.NoError(t, err)

	// the inherited version is itself a property, so it must be evaluated
	// against the merged properties
	p := pom{content: content}
	p.inherit(analysisResult{
		artifact:   artifact{GroupID: "org.example", ArtifactID: "lib", Version: newVersion("${parent.version}")},
		properties: map[string]string{"lib.version": "${parent.version}", "parent.version": "1.2.3"},
	})

	assert.Equal(t, "1.2.3", p.content.Version)
}

func TestPomInheritLiteralVersion(t *testing.T) {
	content, err := parsePom(strings.NewReader(`<project>
  <groupId>org.example</groupId>
  <artifactId>lib</artifactId>
  <version>9.9.9</version>
</project>`))
	require.NoError(t, err)

	p := pom{content: content}
	p.inherit(analysisResult{
		artifact: artifact{GroupID: "org.parent", ArtifactID: "lib", Version: newVersion("1.2.3")},
	})

	// a literal version is kept as-is
	assert.Equal(t, "9.9.9", p.content.Version)
	assert.Equal(t, "org.example", p.content.GroupId)
}

func TestExcludeDep(t *testing.T) {
	exclusions := map[string]struct{}{
		"org.slope:junit": {},
	}

	assert.True(t, excludeDep(exclusions, artifact{GroupID: "org.slope", ArtifactID: "junit"}))
	assert.False(t, excludeDep(exclusions, artifact{GroupID: "org.example", ArtifactID: "lib"}))
	assert.False(t, excludeDep(nil, artifact{GroupID: "org.example", ArtifactID: "lib"}))
}

func TestDepVersion(t *testing.T) {
	uniq := map[string]artifact{
		"org.example:lib": {GroupID: "org.example", ArtifactID: "lib", Version: newVersion("2.0.0")},
	}

	assert.Equal(t, "2.0.0", depVersion("org.example:lib", uniq))
	assert.Equal(t, "", depVersion("org.example:missing", uniq))
}

func TestParseInvalidRootPOM(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "not-a-pom", "pom.xml"))
	require.NoError(t, err)
	defer f.Close()

	p := NewParser(filepath.Join("testdata", "not-a-pom", "pom.xml"))
	_, _, err = p.Parse(f)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse POM")
}

func TestOpenRelativePomErrors(t *testing.T) {
	p := NewParser("testdata/happy/pom.xml", WithOffline(true)).(*parser)

	t.Run("missing path", func(t *testing.T) {
		_, err := p.openRelativePom("testdata/happy/pom.xml", "../nowhere")
		require.Error(t, err)
		assert.ErrorContains(t, err, "no such file or directory")
	})

	t.Run("malformed pom", func(t *testing.T) {
		_, err := p.openRelativePom("testdata/broken-module/pom.xml", "./child")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse the local POM")
	})
}

func TestResolveDepManagementSkipsUnresolvableImports(t *testing.T) {
	p := NewParser("testdata/happy/pom.xml", WithOffline(true)).(*parser)

	depManagement := []pomDependency{
		{GroupID: "org.example", ArtifactID: "managed", Version: "1.0.0"},
		// the imported BOM does not exist, so it is skipped
		{GroupID: "com.example.absent", ArtifactID: "missing-bom", Version: "1.0.0", Scope: "import"},
	}

	got := p.resolveDepManagement(nil, depManagement)
	require.Len(t, got, 1)
	assert.Equal(t, "org.example:managed", got[0].Name())
}

func TestFetchPOMFromRemoteRepository(t *testing.T) {
	paths := []string{"com", "example", "lib", "1.0.0", "lib-1.0.0.pom"}

	t.Run("offline mode", func(t *testing.T) {
		p := NewParser("testdata/happy/pom.xml", WithOffline(true)).(*parser)

		_, err := p.fetchPOMFromRemoteRepository(paths)
		require.Error(t, err)
		assert.ErrorContains(t, err, "offline mode")
	})

	t.Run("unparsable repository URL", func(t *testing.T) {
		p := NewParser("testdata/happy/pom.xml", WithRemoteRepos([]string{"://not a url"})).(*parser)

		_, err := p.fetchPOMFromRemoteRepository(paths)
		require.Error(t, err)
		assert.ErrorContains(t, err, "the POM was not found in remote remoteRepositories")
	})

	t.Run("malformed remote pom", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not xml"))
		}))
		defer ts.Close()

		p := NewParser("testdata/happy/pom.xml", WithRemoteRepos([]string{ts.URL})).(*parser)

		_, err := p.fetchPOMFromRemoteRepository(paths)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse the remote POM")
	})
}
