package stw

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvents(t *testing.T) {
	type testEvent struct {
		Start    time.Duration
		Duration time.Duration
		Type     EventType
	}

	tests := []struct {
		GoVersion   string
		EventCount  int
		CheckEvents map[int]testEvent
	}{
		{
			GoVersion:  "1.19",
			EventCount: 42,
			CheckEvents: map[int]testEvent{
				0: {
					Start:    2323248,
					Duration: 18784,
					Type:     SweepTermination,
				},
				41: {
					Start:    504956960,
					Duration: 89376,
					Type:     MarkTermination,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.GoVersion, func(t *testing.T) {
			// Read the test trace.
			inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", test.GoVersion, "test-encoding-json.trace"))
			require.NoError(t, err)

			// Extract the STW events from the trace
			events, err := Events(bytes.NewReader(inTrace))
			require.NoError(t, err)

			// Assert the number of STW events in the trace.
			require.Equal(t, test.EventCount, len(events))

			// Sort events by start time in ascending order
			sort.Slice(events, func(i, j int) bool {
				return events[i].Start < events[j].Start
			})

			// Check some of the STW events in the trace.
			for i, e := range events {
				want, ok := test.CheckEvents[i]
				if !ok {
					continue
				}
				assert.Equal(t, want.Start, e.Start)
				assert.Equal(t, want.Duration, e.Duration())
				assert.Equal(t, want.Type, e.Type)
			}
		})
	}
}
