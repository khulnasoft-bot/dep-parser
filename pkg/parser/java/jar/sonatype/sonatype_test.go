package sonatype

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/dep-parser/pkg/parser/java/jar"
)

func TestNewDefaults(t *testing.T) {
	s := New()
	assert.Equal(t, baseURL, s.baseURL)
	assert.NotNil(t, s.httpClient)
}

func TestNewWithURL(t *testing.T) {
	s := New(WithURL("https://custom.example.org/search"))
	assert.Equal(t, "https://custom.example.org/search", s.baseURL)
}

func TestNewWithHTTPClient(t *testing.T) {
	client := &http.Client{}
	s := New(WithHTTPClient(client))
	assert.Same(t, client, s.httpClient)
}

func TestExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		q := r.URL.Query()
		assert.Equal(t, `g:"group" AND a:"artifact"`, q.Get("q"))
		assert.Equal(t, "1", q.Get("rows"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"numFound": 1,
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	ok, err := s.Exists("group", "artifact")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestExistsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"numFound": 0,
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	ok, err := s.Exists("group", "artifact")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestExistsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	_, err := s.Exists("group", "artifact")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json decode error")
}

func TestSearchBySHA1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, `1:"deadbeef"`, q.Get("q"))
		assert.Equal(t, "json", q.Get("wt"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"docs": []any{
					map[string]any{
						"id": "g:a:1.0",
						"g":  "group",
						"a":  "artifact",
						"v":  "1.0",
						"p":  "jar",
					},
				},
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	props, err := s.SearchBySHA1("deadbeef")
	require.NoError(t, err)
	assert.Equal(t, jar.Properties{GroupID: "group", ArtifactID: "artifact", Version: "1.0"}, props)
}

func TestSearchBySHA1MultipleDocs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"docs": []any{
					map[string]any{
						"id":           "z:a:1.0",
						"g":            "zgroup",
						"a":            "artifact",
						"v":            "1.0",
						"versionCount": 1,
					},
					map[string]any{
						"id":           "a:a:1.0",
						"g":            "agroup",
						"a":            "artifact",
						"v":            "1.0",
						"versionCount": 1,
					},
				},
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	props, err := s.SearchBySHA1("deadbeef")
	require.NoError(t, err)
	assert.Equal(t, "agroup", props.GroupID)
}

func TestSearchBySHA1NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"docs": []any{},
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	_, err := s.SearchBySHA1("deadbeef")
	require.Error(t, err)
	assert.ErrorIs(t, err, jar.ArtifactNotFoundErr)
}

func TestSearchBySHA1NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	_, err := s.SearchBySHA1("deadbeef")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 404 Not Found")
}

func TestSearchByArtifactID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, `a:"artifact" AND p:"jar"`, q.Get("q"))
		assert.Equal(t, "20", q.Get("rows"))
		assert.Equal(t, "json", q.Get("wt"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"docs": []any{
					map[string]any{
						"id":           "g:a:1.0",
						"g":            "group",
						"a":            "artifact",
						"v":            "1.0",
						"versionCount": 5,
					},
				},
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	got, err := s.SearchByArtifactID("artifact", "ignored")
	require.NoError(t, err)
	assert.Equal(t, "group", got)
}

func TestSearchByArtifactIDMultipleDocs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"docs": []any{
					map[string]any{
						"id":           "g:a:1.0",
						"g":            "group",
						"a":            "artifact",
						"v":            "1.0",
						"versionCount": 1,
					},
					map[string]any{
						"id":           "g:a:2.0",
						"g":            "group",
						"a":            "artifact",
						"v":            "2.0",
						"versionCount": 10,
					},
				},
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	got, err := s.SearchByArtifactID("artifact", "ignored")
	require.NoError(t, err)
	assert.Equal(t, "group", got)
}

func TestSearchByArtifactIDNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"docs": []any{},
			},
		})
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	_, err := s.SearchByArtifactID("artifact", "ignored")
	require.Error(t, err)
	assert.ErrorIs(t, err, jar.ArtifactNotFoundErr)
}

func TestSearchByArtifactIDNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	s := New(WithURL(server.URL), WithHTTPClient(server.Client()))

	_, err := s.SearchByArtifactID("artifact", "ignored")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500 Internal Server Error")
}
