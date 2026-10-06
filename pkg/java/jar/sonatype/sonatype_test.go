package sonatype

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"

	"github.com/khulnasoft/dep-parser/pkg/java/jar"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) Sonatype {
	t.Helper()

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	return New(WithURL(ts.URL), WithHTTPClient(ts.Client()))
}

func TestNewDefaults(t *testing.T) {
	// httptest with retries disabled would be slow, so only assert the defaults
	// point at the public endpoint and a client was created.
	s := New()
	assert.Contains(t, s.baseURL, "search.maven.org")
	assert.NotNil(t, s.httpClient)
}

func TestExists(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		var gotQuery string
		s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query().Get("q")
			_, _ = fmt.Fprint(w, `{"response": {"numFound": 1, "docs": []}}`)
		})

		got, err := s.Exists("org.springframework", "spring-core")
		require.NoError(t, err)
		assert.True(t, got)
		assert.Equal(t, `g:"org.springframework" AND a:"spring-core"`, gotQuery)
	})

	t.Run("not found", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"response": {"numFound": 0, "docs": []}}`)
		})

		got, err := s.Exists("com.example", "nope")
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("invalid json", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `not json`)
		})

		_, err := s.Exists("com.example", "nope")
		assert.ErrorContains(t, err, "json decode error")
	})
}

func TestSearchBySHA1(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		var gotQuery string
		s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query().Get("q")
			// the first doc after sorting by ID wins
			_, _ = fmt.Fprint(w, `{"response": {"numFound": 2, "docs": [
				{"id": "org.springframework:spring-core", "g": "org.springframework", "a": "spring-core", "v": "5.3.3"},
				{"id": "com.example:spring-core", "g": "com.example", "a": "spring-core", "v": "1.0.0"}
			]}}`)
		})

		got, err := s.SearchBySHA1("c666f5bc47eb64ed3bbd13505a26f58be71f33f0")
		require.NoError(t, err)
		assert.Equal(t, jar.Properties{
			GroupID:    "com.example",
			ArtifactID: "spring-core",
			Version:    "1.0.0",
		}, got)
		assert.Equal(t, `1:"c666f5bc47eb64ed3bbd13505a26f58be71f33f0"`, gotQuery)
	})

	t.Run("not found", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"response": {"numFound": 0, "docs": []}}`)
		})

		_, err := s.SearchBySHA1("deadbeef")
		require.ErrorIs(t, err, jar.ArtifactNotFoundErr)
	})

	t.Run("non-200 status", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := s.SearchBySHA1("deadbeef")
		assert.ErrorContains(t, err, "500 Internal Server Error")
	})

	t.Run("invalid json", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `not json`)
		})

		_, err := s.SearchBySHA1("deadbeef")
		assert.ErrorContains(t, err, "json decode error")
	})
}

func TestSearchByArtifactID(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		var gotQuery string
		s := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query().Get("q")
			// the doc with the highest version count wins
			_, _ = fmt.Fprint(w, `{"response": {"numFound": 2, "docs": [
				{"id": "org.springframework:heuristic", "g": "org.springframework", "a": "heuristic", "versionCount": 10},
				{"id": "com.example:heuristic", "g": "com.example", "a": "heuristic", "versionCount": 100}
			]}}`)
		})

		got, err := s.SearchByArtifactID("heuristic", "")
		require.NoError(t, err)
		assert.Equal(t, "com.example", got)
		assert.Equal(t, `a:"heuristic" AND p:"jar"`, gotQuery)
	})

	t.Run("not found", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"response": {"numFound": 0, "docs": []}}`)
		})

		_, err := s.SearchByArtifactID("nope", "")
		require.ErrorIs(t, err, jar.ArtifactNotFoundErr)
	})

	t.Run("non-200 status", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		_, err := s.SearchByArtifactID("nope", "")
		assert.ErrorContains(t, err, "404 Not Found")
	})

	t.Run("invalid json", func(t *testing.T) {
		s := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `not json`)
		})

		_, err := s.SearchByArtifactID("nope", "")
		assert.ErrorContains(t, err, "json decode error")
	})
}

// errRoundTripper fails every request, standing in for a network error.
type errRoundTripper struct{}

func (errRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, xerrors.New("connection refused")
}

func TestInvalidBaseURL(t *testing.T) {
	// an unparsable base URL fails before any request is made
	s := New(WithURL("://not a url"))

	_, err := s.Exists("g", "a")
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to initialize HTTP client")

	_, err = s.SearchBySHA1("deadbeef")
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to initialize HTTP client")

	_, err = s.SearchByArtifactID("a", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to initialize HTTP client")
}

func TestTransportErrors(t *testing.T) {
	client := &http.Client{Transport: errRoundTripper{}}

	tests := []struct {
		name string
		call func(Sonatype) error
		want string
	}{
		{
			name: "Exists",
			call: func(s Sonatype) error {
				_, err := s.Exists("g", "a")
				return err
			},
			want: "http error",
		},
		{
			name: "SearchBySHA1",
			call: func(s Sonatype) error {
				_, err := s.SearchBySHA1("deadbeef")
				return err
			},
			want: "sha1 search error",
		},
		{
			name: "SearchByArtifactID",
			call: func(s Sonatype) error {
				_, err := s.SearchByArtifactID("a", "")
				return err
			},
			want: "artifactID search error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(WithURL("http://example.invalid/solrsearch/select"), WithHTTPClient(client))
			require.Error(t, tt.call(s))
			assert.ErrorContains(t, tt.call(s), tt.want)
		})
	}
}
