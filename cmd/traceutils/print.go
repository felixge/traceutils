package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/felixge/traceutils/pkg/print"
)

func PrintEvents(args []string, filter print.EventFilter) error {
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

	// Print all events to stdout
	stdout := bufio.NewWriter(os.Stdout)
	defer stdout.Flush()
	return print.Events(inFile, stdout, filter)
}

func PrintStacks(args []string, filter print.StackFilter) error {
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

	// Print all events to stdout
	stdout := bufio.NewWriter(os.Stdout)
	defer stdout.Flush()
	return print.Stacks(inFile, stdout, filter)
}
