package encoding

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "trace.bin"))
	require.NoError(t, err)

	p, err := NewDecoder(bytes.NewReader(data))
	require.NoError(t, err)

	var count int
	for {
		e := Event{}
		if err := p.Decode(&e); err != nil {
			require.Equal(t, io.EOF, err)
			break
		}
		count++
	}
	require.Equal(t, 151, count)
}
