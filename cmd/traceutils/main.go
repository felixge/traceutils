package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"

	"github.com/felixge/traceutils/pkg/anonymize"
)

// main is the entry point for the traceutils command line tool.
func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// realMain is a helper function for main that returns an error.
func realMain() error {
	// Set the help text for the command line flags.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traceutils <command> <input> <output>\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  - anonymize: Anonymizes a trace file.\n")
	}

	var (
		cpuProfileF = flag.String("cpuprofile", "", "write cpu profile to file")
		traceF      = flag.String("trace", "", "write trace to file")
	)

	// Parse the command line arguments and run the command using the
	// appropriate function.
	flag.Parse()

	if *cpuProfileF != "" {
		file, err := os.Create(*cpuProfileF)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := pprof.StartCPUProfile(file); err != nil {
			return err
		}
		defer pprof.StopCPUProfile()
	}

	if *traceF != "" {
		file, err := os.Create(*traceF)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := trace.Start(file); err != nil {
			return err
		}
		defer trace.Stop()
	}

	switch cmd := flag.Arg(0); cmd {
	case "anon", "anonymize":
		// Open the input file
		inFile, err := os.Open(flag.Arg(1))
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer inFile.Close()

		// Open the output file
		outFile, err := os.Create(flag.Arg(2))
		if err != nil {
			return fmt.Errorf("failed to open output file: %w", err)
		}
		defer outFile.Close()

		// Anonymize the trace file
		return anonymize.AnonymizeTrace(inFile, outFile)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
