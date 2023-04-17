package anonymize

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixge/traceutils/pkg/encoding"
	"github.com/stretchr/testify/require"
)

// TestAnonymizeTrace tests that we can anonymize an example trace.
func TestAnonymizeTrace(t *testing.T) {
	// Read the test trace.
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "trace.bin"))
	require.NoError(t, err)

	// Anonymize the trace and write it to outTrace.
	var outTrace bytes.Buffer
	require.NoError(t, AnonymizeTrace(bytes.NewReader(inTrace), &outTrace))

	// Create a decoder for the anonymized trace.
	dec := encoding.NewDecoder(bytes.NewReader(outTrace.Bytes()))

	// secretStrings contains strings that appear in inTrace, but that should
	// not appear in the anonymized trace.
	secretStrings := []string{"/Users/", "/felix.geisendoerfer/"}

	xxxPrefix := string(replacement())
	// gotWantedStrings is a map that contains all strings that we expect to
	// appear in the anonymized trace.
	// TODO(fg): Add more strings here.
	gotWantedStrings := map[string]bool{
		xxxPrefix + "/src/runtime/time.go": false,
	}
	for k := range gcMarkWorkerModeStrings {
		gotWantedStrings[k] = false
	}

	// Decode each event and check that it does not contain any secret strings.
	for {
		var ev encoding.Event
		if err := dec.Decode(&ev); err != nil {
			require.Equal(t, io.EOF, err)
			break
		}

		// Check that the string does not contain any secret strings.
		for _, s := range secretStrings {
			require.NotContains(t, string(ev.Str), s)
		}

		if _, ok := gotWantedStrings[string(ev.Str)]; ok {
			gotWantedStrings[string(ev.Str)] = true
		}
	}

	// Check that we got all the strings that we wanted.
	for k, v := range gotWantedStrings {
		require.True(t, v, "did not get string %q", k)
	}

	// TODO: The test trace doesn't contain any EventUserLog events.
	// We should add some code that generates a trace with EventUserLog events and include
	// it in the testdata directory and then use this trace here.
}

// Test_anonymizeString tests the anonymizeString function.
func Test_anonymizeString(t *testing.T) {
	tests := []struct {
		name string
		s    []byte
		want string
	}{
		{
			name: "pkg.func: ok",
			s:    []byte("encoding/json.Marshal"),
			want: "encoding/json.Marshal",
		},

		{
			name: "pkg.func: wrong prefix",
			s:    []byte("my/encoding/json.Marshal"),
			want: "XXX",
		},

		{
			name: "pkg.func: wrong suffix",
			s:    []byte("encoding/json/foo.Marshal"),
			want: "XXX",
		},

		{
			name: "path: ok",
			s:    []byte("/src/runtime/proc.go"),
			want: "XXX/src/runtime/proc.go",
		},

		{
			name: "path: replace prefix",
			s:    []byte("/home/Bob/src/runtime/proc.go"),
			want: "XXX/src/runtime/proc.go",
		},

		{
			name: "path: replace all",
			s:    []byte("/home/Bob/src/runtime/foo/proc.go"),
			want: "XXX.go",
		},

		{
			name: "path: replace all bad prefix",
			s:    []byte("src/runtime/proc.go"),
			want: "XXX",
		},

		{
			name: "path: all tricky",
			s:    []byte("/home/Bob/src/runtime"),
			want: "XXX",
		},

		{
			name: "path: all tricky 2",
			s:    []byte("/home/Bob/src/runtime/"),
			want: "XXX",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(anonymizeString(tt.s))
			if got != tt.want {
				t.Errorf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func BenchmarkAnonymizeTrace(b *testing.B) {
	// Read the test trace.
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "trace.bin"))
	require.NoError(b, err)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Anonymize the trace and write it to io.Discard.
		require.NoError(b, AnonymizeTrace(bytes.NewReader(inTrace), io.Discard))
	}
}

func Test_stdlibPaths(t *testing.T) {
	tests := []struct {
		Path string
		Want bool
	}{
		{Path: "runtime/does-not-exist.go", Want: false},
		{Path: "runtime/proc.go", Want: true},
		{Path: "encoding/json/decode.go", Want: true},
		{Path: "json/decode.go", Want: false},
	}

	for _, tt := range tests {
		t.Run(tt.Path, func(t *testing.T) {
			got := pathInStdLib(tt.Path)
			if got != tt.Want {
				t.Errorf("got=%v want=%v", got, tt.Want)
			}
		})
	}
}
