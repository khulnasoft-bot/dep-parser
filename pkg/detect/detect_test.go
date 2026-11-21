package detect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByName(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{"go.mod", "golang-mod", true},
		{"/any/dir/package-lock.json", "nodejs-npm", true},
		{"./yarn.lock", "nodejs-yarn", true},
		{"Cargo.lock", "rust-cargo", true},
		{"poetry.lock", "python-poetry", true},
		{"app.deps.json", "dotnet", true},
		{"foo.gemspec", "ruby-gemspec", true},
		{"lib.jar", "java-jar", true},
		{"MyApp.csproj", "nuget-props", true},
		{"random.txt", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := ByName(tt.path)
		assert.Equal(t, tt.ok, ok, tt.path)
		assert.Equal(t, tt.want, got, tt.path)
	}
}

func TestDetectNameMatch(t *testing.T) {
	got := Detect("sub/dir/go.sum", nil)
	assert.Equal(t, []string{"golang-sum"}, got)
}

func TestDetectELF(t *testing.T) {
	got := Detect("mystery-bin", []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01})
	assert.Equal(t, []string{"golang-binary", "rust-binary"}, got)
}

func TestDetectMachO(t *testing.T) {
	got := Detect("app", []byte{0xfe, 0xed, 0xfa, 0xcf, 0x00})
	assert.Equal(t, []string{"golang-binary", "rust-binary"}, got)
}

func TestDetectPE(t *testing.T) {
	got := Detect("tool.exe", []byte{'M', 'Z', 0x90, 0x00})
	assert.Equal(t, []string{"golang-binary", "rust-binary"}, got)
}

func TestDetectJar(t *testing.T) {
	got := Detect("unknown.bin", []byte{'P', 'K', 0x03, 0x04, 0x14})
	assert.Equal(t, []string{"java-jar"}, got)
}

func TestDetectJarBySuffix(t *testing.T) {
	got := Detect("libs/foo.jar", nil)
	assert.Equal(t, []string{"java-jar"}, got)
}

func TestDetectNoMatch(t *testing.T) {
	assert.Empty(t, Detect("README.md", []byte("# hello")))
}

func TestDetectNamePlusSniffDedup(t *testing.T) {
	// A file named go.mod that is also ELF-ish: name wins, no dupes.
	got := Detect("go.mod", []byte{0x7f, 'E', 'L', 'F'})
	assert.Equal(t, []string{"golang-mod", "golang-binary", "rust-binary"}, got)
}
