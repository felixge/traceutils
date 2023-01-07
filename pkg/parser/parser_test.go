package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "trace.bin"))
	require.NoError(t, err)

	p, err := NewParser(bytes.NewReader(data))
	require.NoError(t, err)

	for {
		e := Event{}
		if err := p.Parse(&e); err != nil {
			require.Equal(t, io.EOF, err)
			break
		}
		fmt.Printf("e: %v\n", e)
	}
}
