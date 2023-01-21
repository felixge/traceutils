package stw

import (
	"fmt"
	"io"
	"time"

	"github.com/felixge/traceutils/pkg/encoding"
)

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

		switch ev.Type {
		case encoding.EventBatch:
			lastP = ev.Args[0]
			lastTs = time.Duration(ev.Args[1])
		case encoding.EventFrequency:
			ticksPerSec = int64(ev.Args[0])
			if ticksPerSec <= 0 {
				return nil, fmt.Errorf("negative ticksPerSec: %d", ticksPerSec)
			}
		case encoding.EventTimerGoroutine, encoding.EventStack, encoding.EventString:
			// ignore
		default:
			lastTs += time.Duration(ev.Args[0])
			if minTs == 0 || lastTs < minTs {
				minTs = lastTs
			}
		}

		switch ev.Type {
		case encoding.EventGCSTWStart:
			if worldStopped {
				return nil, fmt.Errorf("unexpected EventGCSTWStart: %#v", ev)
			} else {
				events = append(events, &Event{
					Start: lastTs,
					P:     lastP,
				})
				worldStopped = true
			}
		case encoding.EventGCSTWDone:
			if worldStopped {
				event := events[len(events)-1]
				if event.P != lastP {
					return nil, fmt.Errorf("expected P: got=%d want=%d", lastP, event.P)
				}
				event.End = lastTs
				worldStopped = false
			} else {
				return nil, fmt.Errorf("unexpected EventGCSTWDone: %#v", ev)
			}
		}
	}

	if ticksPerSec == 0 {
		return nil, fmt.Errorf("no EventFrequency event")
	}

	freq := 1e9 / float64(ticksPerSec)
	for _, ev := range events {
		ev.Start = time.Duration(float64(ev.Start-minTs) * freq)
		ev.End = time.Duration(float64(ev.End-minTs) * freq)
	}

	return events, nil
}

type Event struct {
	Start time.Duration
	End   time.Duration
	P     uint64
}

func (e Event) Duration() time.Duration {
	return e.End - e.Start
}

type EventType string

var (
	STWFoo EventType = "stwfoo"
)
