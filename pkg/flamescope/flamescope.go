package flamescope

import (
	"fmt"
	"io"
	"math"

	"golang.org/x/exp/trace"
)

// FlameScope reads a trace from r and writes it to w in a format [1] that can
// be visualized by the flamescope tool.
//
// [1] https://github.com/Netflix/flamescope/blob/be26595f1395c32eef71c88a06cb9c8c87f270c2/app/perf/regexp.py
func FlameScope(r io.Reader, w io.Writer) error {
	minTs, cpuSamples, err := getCpuSamples(r)
	if err != nil {
		return err
	}
	// Convert cpu samples to perf script format used by flamescope
	for i := 0; i < len(cpuSamples); i++ {
		// Write the stack trace header with timestamp
		_, _ = fmt.Fprintf(w, "go 0 [0] %f: cpu-clock:\n", float64(cpuSamples[i].Timestamp-minTs)/1e9)
		// Write out the individual stack frames
		frames := cpuSamples[i].Frames
		for j := 0; j < len(frames); j++ {
			_, _ = fmt.Fprintf(w, "\t%x %s (go)\n", frames[j].PC, frames[j].Func)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}
	return nil
}

// cpuSample represents a single cpu sample.
type cpuSample struct {
	Timestamp int64
	Frames    []stackFrame
}

// stackFrame represents a single stack frame.
type stackFrame struct {
	PC   uint64
	Func string
}

func getCpuSamples(f io.Reader) (int64, []cpuSample, error) {
	r, err := trace.NewReader(f)
	if err != nil {
		return 0, nil, err
	}

	minTs := int64(math.MaxInt64)
	cpuSamples := make([]cpuSample, 0, 100)
	var ts int64

	for {
		ev, err := r.ReadEvent()
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, nil, err
		}

		if ev.Kind() == trace.EventStackSample {
			if stk := ev.Stack(); stk != trace.NoStack {
				ts = int64(ev.Time())
				if ts < minTs {
					minTs = ts
				}
				frames := make([]stackFrame, 0, 30)
				for f := range stk.Frames() {
					frames = append(frames, stackFrame{PC: f.PC, Func: f.Func})
				}
				cpuSamples = append(cpuSamples, cpuSample{Timestamp: ts, Frames: frames})
			}
		}
	}
	if len(cpuSamples) <= 0 {
		return 0, nil, fmt.Errorf("not found cpu sample")
	}
	return minTs, cpuSamples, nil
}
