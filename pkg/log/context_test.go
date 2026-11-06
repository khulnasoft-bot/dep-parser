package log

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithContextPrefix(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, ctx, WithContextPrefix(ctx, ""))

	ctx = WithContextPrefix(ctx, "myprefix")
	assert.Equal(t, "[myprefix] ", contextualPrefix(ctx))

	ctx2 := context.Background()
	assert.Equal(t, "", contextualPrefix(ctx2))
}

func TestWithContextAttrs(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, contextualAttrs(ctx))

	ctx = WithContextAttrs(ctx, slog.String("key1", "value1"))
	attrs := contextualAttrs(ctx)
	require.NotNil(t, attrs)
	assert.Len(t, attrs, 1)
	assert.Equal(t, "key1", attrs[0].Key)

	// chained attrs
	ctx = WithContextAttrs(ctx, slog.Int("key2", 2))
	attrs = contextualAttrs(ctx)
	require.NotNil(t, attrs)
	assert.Len(t, attrs, 2)
}

func TestDeferredHandler(t *testing.T) {
	records := new([]deferredRecord)
	h := &DeferredHandler{records: records}

	assert.True(t, h.Enabled(context.Background(), slog.LevelInfo))

	var rec slog.Record
	err := h.Handle(context.Background(), rec)
	require.NoError(t, err)
	assert.Len(t, *records, 1)

	// WithAttrs
	h2 := h.WithAttrs([]slog.Attr{slog.String("k", "v")}).(*DeferredHandler)
	assert.Equal(t, 1, len(h2.attrs))

	// WithGroup panics
	assert.Panics(t, func() { h.WithGroup("g") })

	// Flush
	var buf strings.Builder
	h.Flush(&fakeHandler{out: &buf})
	assert.Nil(t, h.records)
}

type fakeHandler struct {
	out *strings.Builder
}

func (h *fakeHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *fakeHandler) Handle(ctx context.Context, r slog.Record) error {
	h.out.WriteString(r.Message)
	h.out.WriteString("\n")
	return nil
}
func (h *fakeHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *fakeHandler) WithGroup(string) slog.Handler      { return h }
