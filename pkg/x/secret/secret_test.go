package secret

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanEmptyContent(t *testing.T) {
	s := NewScanner(nil)
	result := s.Scan(ScanArgs{Content: nil})
	assert.Empty(t, result.Findings)
}

func TestScanNoPrivateKey(t *testing.T) {
	s := NewScanner(nil)
	result := s.Scan(ScanArgs{Content: strings.NewReader("hello world\nno key here\n")})
	assert.Empty(t, result.Findings)
}

func TestScanPrivateKeyBlock(t *testing.T) {
	s := NewScanner(nil)
	content := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC
-----END PRIVATE KEY-----`
	result := s.Scan(ScanArgs{Content: strings.NewReader(content)})
	assert.Equal(t, 1, len(result.Findings))
	assert.Equal(t, 1, len(result.Findings[0].Code.Lines))
	assert.True(t, result.Findings[0].Code.Lines[0].IsCause)
	assert.Equal(t, 2, result.Findings[0].Code.Lines[0].Number)
	assert.Equal(t, "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC", result.Findings[0].Code.Lines[0].Content)
}

func TestScanUnterminatedPrivateKey(t *testing.T) {
	s := NewScanner(nil)
	content := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC
still in key block`
	result := s.Scan(ScanArgs{Content: strings.NewReader(content)})
	assert.Equal(t, 2, len(result.Findings))
}
