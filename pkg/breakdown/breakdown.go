package breakdown

import (
	"io"

	"github.com/felixge/traceutils/pkg/encoding"
)

// ByEventType reads a trace from r and return a breakdown of it by event type.
func ByEventType(r io.Reader) (EventTypeBreakdown, error) {
	dec := encoding.NewDecoder(r)
	breakdown := make(EventTypeBreakdown)

	var ev encoding.Event
	for {
		start := dec.Offset()
		err := dec.Decode(&ev)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		breakdown[ev.Type] = EventTypeSummary{
			EventType: ev.Type,
			Count:     breakdown[ev.Type].Count + 1,
			// BUG: For the first event we're including the size of the header
			// in the sum here. But that's not a big deal since the header is
			// very small.
			Bytes: breakdown[ev.Type].Bytes + dec.Offset() - start,
		}
	}

	return breakdown, nil
}

// EventTypeBreakdown breaks down the size of a trace by event type.
type EventTypeBreakdown map[encoding.EventType]EventTypeSummary

// EventTypeSummary summarizes the occurence of an event type inside of a trace.
type EventTypeSummary struct {
	// EventType is the type of event.
	EventType encoding.EventType
	// Count is the number of times this event occurred in the trace.
	Count int64
	// Bytes is the amount of data occupied by events of this type in the trace.
	Bytes int64
}
