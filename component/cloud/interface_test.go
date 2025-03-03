package cloud

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSleep(t *testing.T) {
	start := time.Now()
	Sleep(time.Millisecond)
	require.Less(t, time.Since(start), 2*time.Millisecond)
}
