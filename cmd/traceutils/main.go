package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"

	gt "honnef.co/go/gotraceui/trace"

	"github.com/felixge/traceutils/pkg/print"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// main is the entry point for the traceutils command line tool.
func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain() error {
	var (
		rootFlagSet = flag.NewFlagSet("traceutils", flag.ExitOnError)
		cpuProfileF = rootFlagSet.String("cpuprofile", "", "write cpu profile to file")
		traceF      = rootFlagSet.String("trace", "", "write trace to file")

		breakdownFlagSet = flag.NewFlagSet("traceutils breakdown", flag.ExitOnError)

		printFlagSet       = flag.NewFlagSet("traceutils print", flag.ExitOnError)
		printEventsFlagSet = flag.NewFlagSet("traceutils print events", flag.ExitOnError)
		printG             = printEventsFlagSet.Int64("g", -1, "print events concerning this goroutine, -1 means all goroutines")
		printP             = printEventsFlagSet.Int64("p", -1, "print events from this proc, -1 means all procs")
		printMinTs         = printEventsFlagSet.Int64("minTs", 0, "print events with a timestamp >= minTs")
		printMaxTs         = printEventsFlagSet.Int64("maxTs", -1, "print events with a timestamp <= maxTs, -1 means no upper limit")
		printVerbose       = printEventsFlagSet.Bool("v", false, "print stack traces for all events")
		printStacksFlagSet = flag.NewFlagSet("traceutils print stacks", flag.ExitOnError)
		printStackIDs      = printStacksFlagSet.String("ids", "", "print stacks with these ids, comma separated")

		stwFlagSet = flag.NewFlagSet("traceutils stw", flag.ExitOnError)
	)

	anonymize := &ffcli.Command{
		Name:       "anonymize",
		ShortUsage: "traceutils anonymize <input> <output>",
		ShortHelp:  "Anonymizes a trace file.",
		Exec:       func(_ context.Context, args []string) error { return AnonymizeCommand(args) },
	}

	breakdownCSV := &ffcli.Command{
		Name:       "csv",
		ShortUsage: "traceutils breakdown csv <input>",
		ShortHelp:  "Break down a trace by event type, count and bytes as csv.",
		Exec:       func(_ context.Context, args []string) error { return BreakdownCommand(BreakdownCSV, args) },
	}

	breakdownBytes := &ffcli.Command{
		Name:       "bytes",
		ShortUsage: "traceutils breakdown bytes <input>",
		ShortHelp:  "Break down a trace by event type and bytes.",
		Exec:       func(_ context.Context, args []string) error { return BreakdownCommand(BreakdownBytes, args) },
	}

	breakdownCount := &ffcli.Command{
		Name:       "count",
		ShortUsage: "traceutils breakdown count <input>",
		ShortHelp:  "Break down a trace by event type and count.",
		Exec:       func(_ context.Context, args []string) error { return BreakdownCommand(BreakdownCount, args) },
	}

	breakdown := &ffcli.Command{
		Name:        "breakdown",
		ShortUsage:  "traceutils breakdown <subcommand> <input>",
		ShortHelp:   "Break down the contents of a trace.",
		FlagSet:     breakdownFlagSet,
		Subcommands: []*ffcli.Command{breakdownCSV, breakdownBytes, breakdownCount},
		Exec: func(_ context.Context, _ []string) error {
			breakdownFlagSet.Usage()
			return nil
		},
	}

	flamescope := &ffcli.Command{
		Name:       "flamescope",
		ShortUsage: "traceutils flamescope <input> <output>",
		ShortHelp:  "Extract CPU samples from a trace and convert them to a format suitable for FlameScope.",
		Exec:       func(_ context.Context, args []string) error { return FlameScopeCommand(args) },
	}

	printEvents := &ffcli.Command{
		Name:       "events",
		ShortUsage: "traceutils print events <input>",
		ShortHelp:  "Print events contained in the trace.",
		FlagSet:    printEventsFlagSet,
		Exec: func(_ context.Context, args []string) error {
			filter := print.DefaultEventFilter()
			filter.MinTs = gt.Timestamp(*printMinTs)
			filter.MaxTs = gt.Timestamp(*printMaxTs)
			filter.G = *printG
			filter.P = *printP
			filter.Verbose = *printVerbose
			return PrintEvents(args, filter)
		},
	}

	printStacks := &ffcli.Command{
		Name:       "stacks",
		ShortUsage: "traceutils print stacks <input>",
		ShortHelp:  "Print stacks contained in the trace.",
		FlagSet:    printStacksFlagSet,
		Exec: func(_ context.Context, args []string) error {
			filter := print.DefaultStackFilter()
			ids := strings.Split(*printStackIDs, ",")
			for _, idS := range ids {
				if idS == "" {
					continue
				}
				id, err := strconv.ParseUint(idS, 10, 64)
				if err != nil {
					return err
				}
				filter.StackIDs = append(filter.StackIDs, uint32(id))
			}
			return PrintStacks(args, filter)
		},
	}

	print := &ffcli.Command{
		Name:        "print",
		ShortUsage:  "traceutils print <subcommand> <input>",
		ShortHelp:   "Print trace data as plain text.",
		FlagSet:     printFlagSet,
		Subcommands: []*ffcli.Command{printEvents, printStacks},
		Exec: func(_ context.Context, _ []string) error {
			printFlagSet.Usage()
			return nil
		},
	}

	strings := &ffcli.Command{
		Name:       "strings",
		ShortUsage: "traceutils strings <input>",
		ShortHelp:  "Print all strings contained in the trace.",
		Exec:       func(_ context.Context, args []string) error { return StringsCommand(args) },
	}

	stwCSV := &ffcli.Command{
		Name:       "csv",
		ShortUsage: "traceutils stw csv <input>",
		ShortHelp:  "List all stop-the-world events in a trace as csv.",
		Exec:       func(_ context.Context, args []string) error { return STWCommand(STWCSV, args) },
	}

	stwTop := &ffcli.Command{
		Name:       "top",
		ShortUsage: "traceutils stw top <input>",
		ShortHelp:  "List all stop-the-world events in a trace in descending duration order.",
		Exec:       func(_ context.Context, args []string) error { return STWCommand(STWTop, args) },
	}

	stw := &ffcli.Command{
		Name:        "stw",
		ShortUsage:  "traceutils stw <subcommand> <input>",
		ShortHelp:   "List all stop-the-world events in a trace.",
		FlagSet:     stwFlagSet,
		Subcommands: []*ffcli.Command{stwCSV, stwTop},
		Exec: func(_ context.Context, _ []string) error {
			stwFlagSet.Usage()
			return nil
		},
	}

	root := &ffcli.Command{
		ShortUsage:  "traceutils [flags] <subcommand>",
		FlagSet:     rootFlagSet,
		Subcommands: []*ffcli.Command{anonymize, breakdown, print, flamescope, strings, stw},
		Exec: func(_ context.Context, _ []string) error {
			rootFlagSet.Usage()
			return nil
		},
	}

	if err := root.Parse(os.Args[1:]); err != nil {
		return err
	}

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

	return root.Run(context.Background())
}
