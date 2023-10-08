package main

import (
	"fmt"
	"os"

	"github.com/felixge/traceutils/pkg/pprof"
)

func PPROF(args []string, opt pprof.Options) error {
	// Check the number of arguments
	if len(args) != 2 {
		return fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	// Open the input file
	inFile, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	// Open the output file
	outFile, err := os.Create(args[1])
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outFile.Close()

	// Convert trace to pprof
	return pprof.Convert(inFile, outFile, opt)
}
