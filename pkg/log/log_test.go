package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSetLogger(t *testing.T) {
	// the package-level logger is initialized on import
	require.NotNil(t, Logger)

	original := Logger
	t.Cleanup(func() {
		Logger = original
	})

	custom := zap.NewNop().Sugar()
	SetLogger(custom)
	assert.Same(t, custom, Logger)
}
