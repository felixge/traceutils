package stw

import (
	"fmt"
	"io"
	"time"

	"github.com/felixge/traceutils/pkg/encoding"
)

// Events returns a list of all STW events in the given trace.
func Events(r io.Reader) ([]*Event, error) {
	var (
		dec          = encoding.NewDecoder(r) // event decoder
		ev           encoding.Event           // current event
		events       []*Event                 // return events
		ticksPerSec  int64                    // ticks per second
		lastTs       time.Duration            // last timestamp seen
		lastP        uint64                   // last P seen
		minTs        time.Duration            // minimum timestamp
		worldStopped bool                     // true if the world is stopped
	)
	// Read all raw events and turn them into a list of STW events
	for {
		// Read the next event
		err := dec.Decode(&ev)
		if err != nil {
			if err == io.EOF {
				// We're done
				break
			}
			// Failed to decode event
			return nil, err
		}

		// Extract timestamps and P from event
		switch ev.Type {
		case encoding.EventBatch:
			// Every batch belongs to one P
			lastP = ev.Args[0]
			// Each batch has a full timestamp, the remaining events in the
			// batch are relative to this timestamp.
			lastTs = time.Duration(ev.Args[1])
		case encoding.EventFrequency:
			// ticksPerSec is used to convert ticks to nanoseconds
			ticksPerSec = int64(ev.Args[0])
			if ticksPerSec <= 0 {
				return nil, fmt.Errorf("negative ticksPerSec: %d", ticksPerSec)
			}
		case encoding.EventTimerGoroutine, encoding.EventStack, encoding.EventString:
			// Ignore these events, their first argument is not a timestamp
		default:
			// All other events are relative to the last timestamp.
			lastTs += time.Duration(ev.Args[0])
			// Keep track of the minimum timestamp seen.
			// This is technically wrong. The timestamps from EventBatch are
			// what should be used. But we're trying to produce the same results
			// as go tool trace for now.
			if minTs == 0 || lastTs < minTs {
				minTs = lastTs
			}
		}

		// Extract STW events
		switch ev.Type {
		case encoding.EventGCSTWStart:
			if worldStopped {
				return nil, fmt.Errorf("unexpected EventGCSTWStart: %#v", ev)
			} else {
				// Create a new STW event
				event := &Event{Start: lastTs, P: lastP}
				// Determine the type of STW event
				event.Type, err = eventType(dec.Version(), ev.Args[1])
				if err != nil {
					return nil, err
				}
				// Add the event to the list of events
				events = append(events, event)
				// Keep track of the world being stopped
				worldStopped = true
			}
		case encoding.EventGCSTWDone:
			if worldStopped {
				// Find the current STW event
				event := events[len(events)-1]
				// Make sure the P matches, any other P would be a bug in the
				// trace.
				if event.P != lastP {
					return nil, fmt.Errorf("expected P: got=%d want=%d", lastP, event.P)
				}
				// Set the end timestamp
				event.End = lastTs
				// Keep track of the world not beeing stopped anymore
				worldStopped = false
			} else {
				return nil, fmt.Errorf("unexpected EventGCSTWDone: %#v", ev)
			}
		}
	}

	if ticksPerSec == 0 {
		return nil, fmt.Errorf("no EventFrequency event")
	}

	// Convert from ticks to nanoseconds relative to the start of the trace
	freq := 1e9 / float64(ticksPerSec)
	for _, ev := range events {
		ev.Start = time.Duration(float64(ev.Start-minTs) * freq)
		ev.End = time.Duration(float64(ev.End-minTs) * freq)
	}

	return events, nil
}

func eventType(version int, value uint64) (et EventType, err error) {
	if version < 1021 {
		switch value {
		case 0:
			et = MarkTermination
		case 1:
			et = SweepTermination
		default:
			err = fmt.Errorf("unknown STW kind %d", value)
		}
	} else if value < uint64(len(stwReasonGo121[:])) {
		et = stwReasonGo121[value]
	} else {
		err = fmt.Errorf("unknown STW kind %d", value)
	}
	return
}

var stwReasonGo121 = [...]EventType{
	0:  Unknown,
	1:  MarkTermination,
	2:  SweepTermination,
	3:  WriteHeapDump,
	4:  GoroutineProfile,
	5:  GoroutineProfileCleanup,
	6:  AllGoroutinesStackTrace,
	7:  ReadMemStats,
	8:  AllThreadsSyscall,
	9:  GOMAXPROCS,
	10: StartTrace,
	11: StopTrace,
	12: CountPagesInUse,
	13: ReadMetricsSlow,
	14: ReadMemStatsSlow,
	15: PageCachePagesLeaked,
	16: ResetDebugLog,
}

// Event represents a single STW event.
type Event struct {
	// Start is the timestamp when the STW event started.
	Start time.Duration
	// End is the timestamp when the STW event ended.
	End time.Duration
	// Type is the type of the STW event.
	Type EventType
	// P is the P that initiated the STW event.
	P uint64
}

// Duration returns the duration of the STW event.
func (e Event) Duration() time.Duration {
	return e.End - e.Start
}

// EventType is the type of an STW event.
type EventType string

// List of known STW event types.
var (
	Unknown                 EventType = "unknown"
	MarkTermination         EventType = "mark termination"
	SweepTermination        EventType = "sweep termination"
	WriteHeapDump           EventType = "write heap dump"
	GoroutineProfile        EventType = "goroutine profile"
	GoroutineProfileCleanup EventType = "goroutine profile cleanup"
	AllGoroutinesStackTrace EventType = "all goroutines stack trace"
	ReadMemStats            EventType = "read mem stats"
	AllThreadsSyscall       EventType = "AllThreadsSyscall"
	GOMAXPROCS              EventType = "GOMAXPROCS"
	StartTrace              EventType = "start trace"
	StopTrace               EventType = "stop trace"
	CountPagesInUse         EventType = "CountPagesInUse (test)"
	ReadMetricsSlow         EventType = "ReadMetricsSlow (test)"
	ReadMemStatsSlow        EventType = "ReadMemStatsSlow (test)"
	PageCachePagesLeaked    EventType = "PageCachePagesLeaked (test)"
	ResetDebugLog           EventType = "ResetDebugLog (test)"
)
