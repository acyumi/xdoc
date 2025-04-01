package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSleep(t *testing.T) {
	start := time.Now()
	Sleep(time.Millisecond)
	// time.Since也耗时，放宽点到5毫秒
	require.Less(t, time.Since(start), 5*time.Millisecond)
}
