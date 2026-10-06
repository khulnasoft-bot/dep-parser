package jar

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/types"
)

// fakeClient lets each test control what the Maven Central lookups return.
type fakeClient struct {
	exists         bool
	existsErr      error
	sha1           Properties
	sha1Err        error
	artifactID     string
	artifactIDErr  error
	sha1CallCount  int
	existsCallMade bool
}

func (c *fakeClient) Exists(_, _ string) (bool, error) {
	c.existsCallMade = true
	return c.exists, c.existsErr
}

func (c *fakeClient) SearchBySHA1(string) (Properties, error) {
	c.sha1CallCount++
	return c.sha1, c.sha1Err
}

func (c *fakeClient) SearchByArtifactID(_, _ string) (string, error) {
	return c.artifactID, c.artifactIDErr
}

// zipWith builds an in-memory zip archive from a name -> content map.
func zipWith(t *testing.T, files map[string]string) ([]byte, *bytes.Reader) {
	t.Helper()

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = io.WriteString(f, content)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	return buf.Bytes(), bytes.NewReader(buf.Bytes())
}

const validManifest = `Manifest-Version: 1.0
Implementation-Version: 1.0.0
Implementation-Title: example
Implementation-Vendor-Id: org.example
`

func TestParseArtifact(t *testing.T) {
	t.Run("pom.properties matching the file name wins over MANIFEST.MF", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": validManifest,
			"META-INF/maven/org.example/example/pom.properties": `groupId=org.example
artifactId=example
version=1.0.0
`,
		})

		c := &fakeClient{}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		got, gotDeps, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		assert.Nil(t, gotDeps)
		require.Len(t, got, 1)
		assert.Equal(t, "org.example:example", got[0].Name)
		assert.Equal(t, "1.0.0", got[0].Version)
		// pom.properties was authoritative, so no remote lookup is needed
		assert.False(t, c.existsCallMade)
		assert.Zero(t, c.sha1CallCount)
	})

	t.Run("pom.properties for a different artifact does not suppress MANIFEST.MF", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": validManifest,
			"META-INF/maven/org.example/other/pom.properties": `groupId=org.example
artifactId=other
version=2.0.0
`,
		})

		c := &fakeClient{exists: true}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.True(t, c.existsCallMade)
	})

	t.Run("unreadable inner jar is skipped", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": validManifest,
			// valid zip entry, but the content is not a zip archive
			"WEB-INF/lib/not-really-a.jar": "this is not a zip archive",
		})

		c := &fakeClient{exists: true}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "org.example:example", got[0].Name)
	})

	t.Run("inner jar libraries are collected", func(t *testing.T) {
		innerContent, _ := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": `Manifest-Version: 1.0
Implementation-Version: 2.0.0
Implementation-Title: inner
Implementation-Vendor-Id: org.inner
`,
		})

		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF":        validManifest,
			"WEB-INF/lib/inner-2.0.0.jar": string(innerContent),
		})

		c := &fakeClient{exists: true}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)

		names := make([]string, 0, len(got))
		for _, l := range got {
			names = append(names, l.Name)
		}
		assert.ElementsMatch(t, []string{"org.example:example", "org.inner:inner"}, names)
	})

	t.Run("offline with an unusable manifest returns what was collected", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\nBundle-Name: %placeholder\n",
		})

		c := &fakeClient{}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), offline: true, client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.False(t, c.existsCallMade)
	})

	t.Run("SHA-1 lookup resolves the artifact", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		})

		c := &fakeClient{sha1: Properties{GroupID: "com.example", ArtifactID: "found", Version: "3.0.0"}}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "com.example:found", got[0].Name)
		assert.Equal(t, "3.0.0", got[0].Version)
		assert.Equal(t, 1, c.sha1CallCount)
	})

	t.Run("SHA-1 lookup failure other than not-found aborts", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{
			"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
		})

		c := &fakeClient{sha1Err: xerrors.New("connection refused")}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		_, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to search by SHA1")
	})

	t.Run("artifactID lookup resolves the group when the file name has one", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{})

		c := &fakeClient{
			sha1Err:    ArtifactNotFoundErr,
			artifactID: "com.example",
		}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "com.example:example", got[0].Name)
	})

	t.Run("artifactID lookup failure other than not-found aborts", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{})

		c := &fakeClient{
			sha1Err:       ArtifactNotFoundErr,
			artifactIDErr: xerrors.New("connection refused"),
		}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		_, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to search by artifact id")
	})

	t.Run("unparseable file name stops before the artifactID lookup", func(t *testing.T) {
		content, r := zipWith(t, map[string]string{})

		c := &fakeClient{sha1Err: ArtifactNotFoundErr}
		p := &Parser{rootFilePath: "no-version-here.jar", size: int64(len(content)), client: c}

		got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("not a zip archive", func(t *testing.T) {
		content := []byte("definitely not a zip")
		r := bytes.NewReader(content)
		c := &fakeClient{}
		p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

		_, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
		require.Error(t, err)
		assert.ErrorContains(t, err, "zip error")
	})
}

// oversizedLine produces content whose single line exceeds bufio.Scanner's
// default token size, which surfaces the scanner error paths.
func oversizedLine() string {
	return strings.Repeat("a", 128*1024)
}

func TestParsePomPropertiesScanError(t *testing.T) {
	content, r := zipWith(t, map[string]string{
		"META-INF/maven/org.example/example/pom.properties": oversizedLine(),
	})

	c := &fakeClient{}
	p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

	_, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
	require.Error(t, err)
	assert.ErrorContains(t, err, "scan error")
}

func TestParseManifestScanError(t *testing.T) {
	content, r := zipWith(t, map[string]string{
		"META-INF/MANIFEST.MF": oversizedLine(),
	})

	c := &fakeClient{}
	p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

	_, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
	require.Error(t, err)
	assert.ErrorContains(t, err, "scan error")
}

func TestParseWrapsArtifactErrors(t *testing.T) {
	content := []byte("not a zip")
	r := bytes.NewReader(content)

	p := NewParser(&fakeClient{}, WithFilePath("example-1.0.0.jar"), WithSize(int64(len(content))))

	_, _, err := p.Parse(r)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to parse example-1.0.0.jar")
	assert.ErrorContains(t, err, "zip error")
}

// errOnReadSeeker fails on a chosen operation so the io error branches run.
// The reader is held as a named field rather than embedded so that io.Copy
// cannot take the WriteTo fast path and bypass the failing Read.
type errOnReadSeeker struct {
	src      *bytes.Reader
	failSeek bool
	failRead bool
}

func (r *errOnReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if r.failSeek {
		return 0, xerrors.New("seek boom")
	}
	return r.src.Seek(offset, whence)
}

func (r *errOnReadSeeker) Read(p []byte) (int, error) {
	if r.failRead {
		return 0, xerrors.New("read boom")
	}
	return r.src.Read(p)
}

func (r *errOnReadSeeker) ReadAt(p []byte, off int64) (int, error) {
	return r.src.ReadAt(p, off)
}

func TestSearchBySHA1Errors(t *testing.T) {
	content, _ := zipWith(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
	})

	t.Run("seek failure", func(t *testing.T) {
		r := &errOnReadSeeker{src: bytes.NewReader(content), failSeek: true}
		p := &Parser{client: &fakeClient{}}

		_, err := p.searchBySHA1(r, "example-1.0.0.jar")
		require.Error(t, err)
		assert.ErrorContains(t, err, "file seek error")
	})

	t.Run("read failure while hashing", func(t *testing.T) {
		r := &errOnReadSeeker{src: bytes.NewReader(content), failRead: true}
		p := &Parser{client: &fakeClient{}}

		_, err := p.searchBySHA1(r, "example-1.0.0.jar")
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to calculate SHA-1")
	})

	t.Run("client error is propagated and file path is set on success", func(t *testing.T) {
		r := bytes.NewReader(content)
		c := &fakeClient{sha1: Properties{GroupID: "g", ArtifactID: "a", Version: "v"}}
		p := &Parser{client: c}

		got, err := p.searchBySHA1(r, "example-1.0.0.jar")
		require.NoError(t, err)
		assert.Equal(t, "example-1.0.0.jar", got.FilePath)
	})
}

func TestParseInnerJarOpenError(t *testing.T) {
	// A directory entry named like an artifact: it opens, but there is nothing
	// to read, so parsing the inner "jar" fails and the caller skips it.
	content, r := zipWith(t, map[string]string{
		"META-INF/MANIFEST.MF":         validManifest,
		"WEB-INF/lib/empty-1.0.0.jar/": "",
	})

	c := &fakeClient{exists: true}
	p := &Parser{rootFilePath: "example-1.0.0.jar", size: int64(len(content)), client: c}

	got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "org.example:example", got[0].Name)
}

func TestParseInnerJarTempFilePathIsRecorded(t *testing.T) {
	innerContent, _ := zipWith(t, map[string]string{
		"META-INF/maven/org.inner/inner/pom.properties": `groupId=org.inner
artifactId=inner
version=2.0.0
`,
	})

	content, r := zipWith(t, map[string]string{
		"WEB-INF/lib/inner-2.0.0.jar": string(innerContent),
	})

	// the outer artifact has no parsable name, so nothing but the inner
	// jar's library comes back
	c := &fakeClient{sha1Err: ArtifactNotFoundErr}
	p := &Parser{rootFilePath: "outer.jar", size: int64(len(content)), client: c}

	got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
	require.NoError(t, err)
	require.Len(t, got, 1)
	// nested jars carry the path of the file that contained them
	assert.Equal(t, "outer.jar/WEB-INF/lib/inner-2.0.0.jar", got[0].FilePath)
}

func TestParseArtifactTypesAssertion(t *testing.T) {
	// guards the helper types used above against accidental drift
	var c Client = &fakeClient{}
	require.Implements(t, (*Client)(nil), c)
	require.NotNil(t, types.Library{})
}

func TestParseInnerJarTempFileError(t *testing.T) {
	innerContent, _ := zipWith(t, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n",
	})

	content, r := zipWith(t, map[string]string{
		"WEB-INF/lib/inner-2.0.0.jar": string(innerContent),
	})

	// an unusable temp dir makes os.CreateTemp fail, so the inner jar is
	// skipped rather than failing the whole artifact
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does-not-exist"))

	c := &fakeClient{sha1Err: ArtifactNotFoundErr}
	p := &Parser{rootFilePath: "outer.jar", size: int64(len(content)), client: c}

	got, _, err := p.parseArtifact(p.rootFilePath, p.size, r)
	require.NoError(t, err)
	assert.Empty(t, got)
}
