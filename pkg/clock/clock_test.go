package clock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithNow(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := With(context.Background(), t1)
	assert.Equal(t, t1, Now(ctx))
}

func TestClockFromContext(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := With(context.Background(), t1)
	assert.Equal(t, t1, Clock(ctx).Now())
}

func TestClockDefaultRealClock(t *testing.T) {
	now := Now(context.Background())
	assert.True(t, now.Before(time.Now().Add(time.Second)))
	assert.True(t, now.After(time.Now().Add(-time.Second)))
}
