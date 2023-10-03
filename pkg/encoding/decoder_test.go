package encoding

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDecoder tests that we can decode a trace without errors.
// This does not check the correctness of the events, just that we can decode them.
// The correctness of the events is checked in the TestDecodeEncode test.
func TestDecoder(t *testing.T) {
	// Read the test trace.
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "1.19", "trace.bin"))
	require.NoError(t, err)

	// Create a decoder
	dec := NewDecoder(bytes.NewReader(data))

	// Decode each event and count them.
	var count int
	for {
		e := Event{}
		if err := dec.Decode(&e); err != nil {
			require.Equal(t, io.EOF, err)
			break
		}
		count++
	}
	// Check that we decoded the correct number of events.
	require.Equal(t, 151, count)
}

// TestDecodeEncode tests that we can decode and encode a trace.
// This is a round-trip test that checks that the encoded trace is the same as
// the original.
func TestDecodeEncode(t *testing.T) {
	tests := []struct {
		Trace string
	}{
		{"trace.bin"},
		{"staticcheck.trace"},
	}

	for _, test := range tests {
		t.Run(test.Trace, func(t *testing.T) {
			// Read the test trace.
			inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "1.19", test.Trace))
			require.NoError(t, err)

			// Create a decoder
			dec := NewDecoder(bytes.NewReader(inTrace))

			// Create an encoder
			var outTrace bytes.Buffer
			enc := NewEncoder(&outTrace)

			// Decode and encode each event.
			for i := 0; ; i++ {
				beforeLen := outTrace.Len()
				e := Event{}
				if err := dec.Decode(&e); err != nil {
					require.Equal(t, io.EOF, err)
					break
				}
				require.NoError(t, enc.Encode(&e))

				// Check output after each event to understand errors without having to
				// diff the whole binary output.
				gotEncoded := outTrace.Bytes()[beforeLen:]
				wantEncoded := inTrace[beforeLen : beforeLen+len(gotEncoded)]
				require.Equal(t, wantEncoded, gotEncoded, "failed to encode event %d: %v", i, e)
			}

			// Check that the length of the encoded trace is the same as the original.
			require.Equal(t, len(inTrace), outTrace.Len())
			// Check that the encoded trace is the same as the original.
			require.Equal(t, inTrace, outTrace.Bytes())
			// Check that the offset is now equal to the size of the input.
			require.Equal(t, int64(len(inTrace)), dec.Offset())
		})
	}
}
