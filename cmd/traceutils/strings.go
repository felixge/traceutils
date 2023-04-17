package main

import (
	"fmt"
	"io"
	"os"

	"github.com/felixge/traceutils/pkg/encoding"
)

func StringsCommand(args []string) error {
	// Check the number of arguments
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	// Open the input file
	inFile, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	dec := encoding.NewDecoder(inFile)

	// Obfuscate all string events
	var ev encoding.Event
	for {
		// Decode event
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Print the string
		if len(ev.Str) > 0 {
			fmt.Println(string(ev.Str))
		}
	}
}
