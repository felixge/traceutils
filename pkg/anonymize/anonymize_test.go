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

func TestBytes(t *testing.T) {
	allowed := []string{"runtime", "encoding/json"}
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
			want: "xx/xxxxxxxx/xxxx.Xxxxxxx",
		},

		{
			name: "pkg.func: wrong suffix",
			s:    []byte("encoding/json/foo.Marshal"),
			want: "xxxxxxxx/xxxx/xxx.Xxxxxxx",
		},

		{
			name: "path: ok",
			s:    []byte("/src/runtime/proc.go"),
			want: "/src/runtime/proc.go",
		},

		{
			name: "path: replace prefix",
			s:    []byte("/home/Bob/src/runtime/proc.go"),
			want: "/xxxx/Xxx/src/runtime/proc.go",
		},

		{
			name: "path: replace all",
			s:    []byte("/home/Bob/src/runtime/foo/proc.go"),
			want: "/xxxx/Xxx/xxx/xxxxxxx/xxx/xxxx.go",
		},

		{
			name: "path: all tricky",
			s:    []byte("/home/Bob/src/runtime"),
			want: "/xxxx/Xxx/xxx/xxxxxxx",
		},

		{
			name: "path: all tricky 2",
			s:    []byte("/home/Bob/src/runtime/"),
			want: "/xxxx/Xxx/xxx/xxxxxxx/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anonymizeString(tt.s, allowed)
			if got := string(tt.s); got != tt.want {
				t.Errorf("got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestAnonymizeTrace(t *testing.T) {
	// Read the test trace.
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "trace.bin"))
	require.NoError(t, err)

	// Anonymize the trace and write it to outTrace.
	var outTrace bytes.Buffer
	require.NoError(t, AnonymizeTrace(bytes.NewReader(inTrace), &outTrace))

	// Create a decoder for the anonymized trace.
	dec, err := encoding.NewDecoder(bytes.NewReader(outTrace.Bytes()))
	require.NoError(t, err)

	secretStrings := []string{"/Users/", "/felix.geisendoerfer/"}
	for {
		var ev encoding.Event
		if err := dec.Decode(&ev); err != nil {
			require.Equal(t, io.EOF, err)
			break
		}

		if ev.Type == encoding.EventString {
			// Check that the string does not contain any secret strings.
			for _, s := range secretStrings {
				require.NotContains(t, string(ev.Str), s)
			}
		}
	}
}
