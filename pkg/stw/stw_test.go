package stw

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvents(t *testing.T) {
	// Read the test trace.
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test-encoding-json.trace"))
	require.NoError(t, err)

	// Extract the STW events from the trace
	events, err := Events(bytes.NewReader(inTrace))
	require.NoError(t, err)

	// Assert the number of STW events in the trace.
	require.Equal(t, 42, len(events))

	// Sort events by start time in ascending order
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start < events[j].Start
	})

	// Validate first event
	first := events[0]
	require.Equal(t, time.Duration(2323248), first.Start)
	require.Equal(t, time.Duration(18784), first.Duration())
	require.Equal(t, SweepTermination, first.Type)

	// Validate last event
	last := events[len(events)-1]
	require.Equal(t, time.Duration(504956960), last.Start)
	require.Equal(t, time.Duration(89376), last.Duration())
	require.Equal(t, MarkTermination, last.Type)
}
