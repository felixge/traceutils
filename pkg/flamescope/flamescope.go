package flamescope

import (
	"errors"
	"fmt"
	"io"

	"github.com/felixge/traceutils/pkg/encoding"
)

// FlameScope reads a trace from r and writes it to w in a format [1] that can
// be visualized by the flamescope tool.
//
// [1] https://github.com/Netflix/flamescope/blob/be26595f1395c32eef71c88a06cb9c8c87f270c2/app/perf/regexp.py
func FlameScope(r io.Reader, w io.Writer) (err error) {
	var (
		dec         = encoding.NewDecoder(r)        // event decoder
		ev          encoding.Event                  // current event
		ticksPerSec int64                           // ticks per second
		minTS       uint64                          // minimum timestamp
		cpuSamples  []cpuSample                     // cpu samples
		stacks      = make(map[uint64][]stackFrame) // stack id : stack frames
		strings     = make(map[uint64]string)       // string id : string
	)
	// Read all raw events
	for {
		// Read the next event
		err := dec.Decode(&ev)
		if err != nil {
			if err == io.EOF {
				// We're done
				break
			}
			// Failed to decode event
			return err
		}

		switch ev.Type {
		// CPU profiling sample [timestamp, real timestamp, real P id (-1 when absent), goroutine id, stack]
		case encoding.EventCPUSample:
			sample := cpuSample{Timestamp: ev.Args[1], StackID: ev.Args[4]}
			cpuSamples = append(cpuSamples, sample)
			if minTS == 0 || sample.Timestamp < minTS {
				minTS = sample.Timestamp
			}
		// stack [stack id, number of PCs, array of {PC, func string ID, file string ID, line}]
		case encoding.EventStack:
			stackID, numFrames := ev.Args[0], ev.Args[1]
			frames := make([]stackFrame, numFrames)
			args := ev.Args[2:]
			for i := 0; i < len(frames); i++ {
				frames[i].PC = args[i*4+0]
				frames[i].FuncID = args[i*4+1]
				frames[i].FileID = args[i*4+2]
				frames[i].Line = args[i*4+3]
			}
			stacks[stackID] = frames
		// string dictionary entry [ID, length, string]
		case encoding.EventString:
			strings[ev.Args[0]] = string(ev.Str)
		case encoding.EventFrequency:
			// ticksPerSec is used to convert ticks to nanoseconds
			ticksPerSec = int64(ev.Args[0])
		}
	}

	if ticksPerSec <= 0 {
		return errors.New("missing or bad EventFrequency event")
	}

	// Convert cpu samples to perf script format used by flamescope
	for _, cs := range cpuSamples {
		// Convert ticks to seconds
		ts := float64(cs.Timestamp-minTS) / float64(ticksPerSec)
		// Write the stack trace header with timestamp
		fmt.Fprintf(w, "go 0 [0] %f: cpu-clock:\n", ts)
		// Write out the individual stack frames
		stack := stacks[cs.StackID]
		for _, sf := range stack {
			fn := strings[sf.FuncID]
			fmt.Fprintf(w, "\t%x %s (go)\n", sf.PC, fn)
		}
		fmt.Fprintf(w, "\n")
	}
	return nil
}

// cpuSample represents a single cpu sample.
type cpuSample struct {
	Timestamp uint64
	StackID   uint64
}

// stackFrame represents a single stack frame.
type stackFrame struct {
	PC     uint64
	FuncID uint64
	FileID uint64
	Line   uint64
}
