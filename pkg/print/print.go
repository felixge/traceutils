package print

import (
	"fmt"
	"io"

	"slices"

	"honnef.co/go/gotraceui/trace"
)

// DefaultEventFilter returns a filter that matches all events.
func DefaultEventFilter() EventFilter {
	return EventFilter{MaxTs: -1, P: -1, G: -1}
}

// EventFilter is used to filter events.
type EventFilter struct {
	// MinTs prints events with a timestamp >= MinTs. The unit is nanoseconds.
	MinTs trace.Timestamp
	// MaxTS prints events with a timestamp <= MaxTs. The unit is nanoseconds.
	// If MaxTs is -1, there is no upper limit.
	MaxTs trace.Timestamp
	// Only prints events from this proc. If P is -1 events from all procs are
	// printed.
	P int64
	// Only prints events from this goroutine. If G is -1 events from all
	// goroutines are printed.
	G int64
	// Verbose prints stack traces for all events.
	Verbose bool
	// StackIDs prints events with these stack ids. If StackIDs is empty, all
	// events are printed.
	StackIDs []uint32
}

// Events prints all events contained in r that match the given filter to w.
func Events(r io.Reader, w io.Writer, filter EventFilter) error {
	trace, err := trace.Parse(r, nil)
	if err != nil {
		return err
	}
	for _, e := range trace.Events {
		if !matchMinTs(e, filter.MinTs) ||
			!matchMaxTs(e, filter.MaxTs) ||
			!matchP(e, filter.P) ||
			!matchG(e, filter.G) ||
			!matchStackIDs(e, filter.StackIDs) {
			continue
		}
		printEvent(w, trace, e)
		io.WriteString(w, "\n")
		if filter.Verbose {
			printStacks(w, trace, e)
		}
	}
	return nil
}

// matchMinTs returns true if e is >= minTs.
func matchMinTs(e trace.Event, minTs trace.Timestamp) bool {
	return e.Ts >= minTs
}

// matchMaxTs returns true if e is <= maxTs or maxTs is -1.
func matchMaxTs(e trace.Event, maxTs trace.Timestamp) bool {
	return maxTs == -1 || e.Ts <= maxTs
}

// matchP returns true if e belongs to proc p or p is -1.
func matchP(e trace.Event, p int64) bool {
	return p == -1 || e.P == int32(p)
}

// matchG returns true if e is concerning goroutine g.
func matchG(e trace.Event, g int64) bool {
	if g == -1 || e.G == uint64(g) {
		return true
	}
	desc := &trace.EventDescriptions[e.Type]
	for i, v := range e.Args[:] {
		if i < len(desc.Args) && desc.Args[i] == "g" {
			if v == uint64(g) {
				return true
			}
		}
	}
	return false
}

func matchStackIDs(e trace.Event, stackIDs []uint32) bool {
	if len(stackIDs) == 0 || slices.Contains(stackIDs, e.StkID) {
		return true
	}
	desc := &trace.EventDescriptions[e.Type]
	for i, v := range e.Args[:] {
		if i < len(desc.Args) && desc.Args[i] == "stack" {
			if slices.Contains(stackIDs, uint32(v)) {
				return true
			}
		}
	}
	return false
}

// printEvent prints a single event to w.
func printEvent(w io.Writer, t trace.Trace, e trace.Event) {
	io.WriteString(w, e.String())
	switch e.Type {
	case trace.EvUserTaskCreate:
		io.WriteString(w, " category=")
		io.WriteString(w, t.Strings[e.Args[2]])

	case trace.EvUserLog:
		io.WriteString(w, " category=")
		io.WriteString(w, t.Strings[e.Args[1]])
		io.WriteString(w, " message=")
		io.WriteString(w, t.Strings[e.Args[3]])
	}
}

// printEvent prints a single event to w.
func printStacks(w io.Writer, t trace.Trace, e trace.Event) {
	var stackIDs []uint32
	if e.StkID != 0 {
		stackIDs = append(stackIDs, e.StkID)
	}

	desc := &trace.EventDescriptions[e.Type]
	for i, v := range e.Args[:] {
		if i < len(desc.Args) && desc.Args[i] == "stack" {
			stackIDs = append(stackIDs, uint32(v))
		}
	}

	for i, stackID := range stackIDs {
		if i > 0 {
			io.WriteString(w, "\n")
		}
		printStack(w, t, stackID)
	}
}

// DefaultStackFilter returns a filter that matches all stacks.
func DefaultStackFilter() StackFilter {
	return StackFilter{}
}

// StackFilter is used to filter stacks.
type StackFilter struct {
	StackIDs []uint32
}

// Stacks prints all stacks contained in r that match the given filter to w.
func Stacks(r io.Reader, w io.Writer, filter StackFilter) error {
	trace, err := trace.Parse(r, nil)
	if err != nil {
		return err
	}

	var stackIDs []uint32
	for stackID := range trace.Stacks {
		stackIDs = append(stackIDs, stackID)
	}
	slices.Sort(stackIDs)

	n := 0
	for _, id := range stackIDs {
		if !matchStacks(id, filter.StackIDs) {
			continue
		}
		if n > 0 {
			io.WriteString(w, "\n")
		}
		n++
		printStack(w, trace, id)
	}
	return nil
}

// matchStacks returns true if id is contained in ids or ids is empty.
func matchStacks(id uint32, ids []uint32) bool {
	return len(ids) == 0 || slices.Contains(ids, id)
}

// printStack prints a single stack to w.
func printStack(w io.Writer, t trace.Trace, id uint32) {
	pcs := t.Stacks[id]
	fmt.Fprintf(w, "stack %d:\n", id)
	for _, pc := range pcs {
		frame := t.PCs[pc]
		fmt.Fprintf(w, "\t%s()\n\t\t%s:%d\n", frame.Fn, frame.File, frame.Line)
	}
}
