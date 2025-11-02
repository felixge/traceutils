package flamescope

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/require"
)

func TestFlameScope(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "go1.19",
			version: "1.19",
		},
		{
			name:    "go1.25",
			version: "1.25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read the test trace.
			inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", tt.version, "test-encoding-json.trace"))
			require.NoError(t, err)

			// Extract the STW events from the trace
			var out bytes.Buffer
			require.NoError(t, FlameScope(bytes.NewReader(inTrace), &out))

			// Compare the output to the expected output.
			snaps.MatchSnapshot(t, out.String())
		})
	}
}
