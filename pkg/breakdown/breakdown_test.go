package breakdown

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixge/traceutils/pkg/encoding"
	"github.com/stretchr/testify/require"
)

func TestByEventType(t *testing.T) {
	// Read the test trace.
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "trace.bin"))
	require.NoError(t, err)

	// Break down the trace by event type.
	breakdown, err := ByEventType(bytes.NewReader(inTrace))
	require.NoError(t, err)

	// Assert the number of event types in the trace.
	require.Equal(t, 18, len(breakdown))
	// Assert the sum of all event bytes equals the size of the input trace.
	var size int64
	for _, summary := range breakdown {
		size += summary.Bytes
	}
	require.Equal(t, int64(len(inTrace)), size)
	// Spot check of type of event
	require.Equal(t, breakdown[encoding.EventString].EventType, encoding.EventString)
	require.Equal(t, breakdown[encoding.EventString].Count, int64(41))
	require.Equal(t, breakdown[encoding.EventString].Bytes, int64(1694))
}
