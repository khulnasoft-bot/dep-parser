package pom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSettings(t *testing.T) {
	writeSettings := func(t *testing.T, dir, localRepo string) {
		t.Helper()

		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".m2"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "conf"), 0o755))

		if localRepo != "" {
			require.NoError(t, os.WriteFile(
				filepath.Join(dir, ".m2", "settings.xml"),
				[]byte(`<settings><localRepository>`+localRepo+`</localRepository></settings>`), 0o600))
			require.NoError(t, os.WriteFile(
				filepath.Join(dir, "conf", "settings.xml"),
				[]byte(`<settings><localRepository>global-repo</localRepository></settings>`), 0o600))
		}
	}

	t.Run("user settings win", func(t *testing.T) {
		home := t.TempDir()
		global := t.TempDir()
		writeSettings(t, home, "user-repo")
		writeSettings(t, global, "global-repo")

		t.Setenv("HOME", home)
		t.Setenv("MAVEN_HOME", global)

		assert.Equal(t, "user-repo", readSettings().LocalRepository)
	})

	t.Run("global settings are used when the user has none", func(t *testing.T) {
		home := t.TempDir()
		global := t.TempDir()
		writeSettings(t, home, "")
		writeSettings(t, global, "global-repo")

		t.Setenv("HOME", home)
		t.Setenv("MAVEN_HOME", global)

		assert.Equal(t, "global-repo", readSettings().LocalRepository)
	})

	t.Run("no settings at all", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("MAVEN_HOME", t.TempDir())

		assert.Equal(t, settings{}, readSettings())
	})

	t.Run("unreadable settings are ignored", func(t *testing.T) {
		home := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(home, ".m2"), 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(home, ".m2", "settings.xml"), []byte("not xml"), 0o600))

		t.Setenv("HOME", home)
		t.Setenv("MAVEN_HOME", t.TempDir())

		assert.Equal(t, settings{}, readSettings())
	})
}

func TestOpenSettings(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := openSettings(filepath.Join(t.TempDir(), "absent.xml"))
		require.Error(t, err)
	})

	t.Run("malformed xml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.xml")
		require.NoError(t, os.WriteFile(path, []byte("not xml"), 0o600))

		_, err := openSettings(path)
		require.Error(t, err)
	})

	t.Run("no local repository configured", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.xml")
		require.NoError(t, os.WriteFile(path, []byte(`<settings></settings>`), 0o600))

		got, err := openSettings(path)
		require.NoError(t, err)
		assert.Equal(t, settings{}, got)
	})
}

func TestIsDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	got, err := isDirectory(dir)
	require.NoError(t, err)
	assert.True(t, got)

	got, err = isDirectory(file)
	require.NoError(t, err)
	assert.False(t, got)

	_, err = isDirectory(filepath.Join(dir, "absent"))
	require.Error(t, err)
}

func TestIsProperty(t *testing.T) {
	assert.True(t, isProperty("${version}"))
	assert.False(t, isProperty("1.0.0"))
	assert.False(t, isProperty(""))
	assert.False(t, isProperty("${version"))
	assert.False(t, isProperty("version}"))
}

func TestPackageID(t *testing.T) {
	assert.Equal(t, "org.example:lib:1.0.0", packageID("org.example:lib", "1.0.0"))
}
