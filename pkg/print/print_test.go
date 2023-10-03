package print

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"honnef.co/go/gotraceui/trace"
)

func TestEvents(t *testing.T) {
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "1.19", "trace.bin"))
	require.NoError(t, err)

	t.Run("Default Filter", func(t *testing.T) {
		out := events(t, inTrace, DefaultEventFilter())
		assert.True(t, containsTextEvent(out, 0, trace.EvGoCreate))
		assert.True(t, containsTextEvent(out, 159632, trace.EvProcStart))
		assert.True(t, containsTextEvent(out, 1001309920, trace.EvGoSched))

		require.False(t, strings.Contains(out, "stack 8:"))
		require.False(t, strings.Contains(out, "runtime/trace.Start.func1()"))
	})

	t.Run("Time Filter", func(t *testing.T) {
		f := DefaultEventFilter()
		f.MinTs = 159632
		f.MaxTs = 159632
		out := events(t, inTrace, f)
		assert.False(t, containsTextEvent(out, 0, trace.EvGoCreate))
		assert.True(t, containsTextEvent(out, 159632, trace.EvProcStart))
		assert.False(t, containsTextEvent(out, 1001309920, trace.EvGoSched))
	})

	t.Run("P Filter", func(t *testing.T) {
		f := DefaultEventFilter()
		f.P = 9
		out := events(t, inTrace, f)
		assert.False(t, containsTextEvent(out, 0, trace.EvGoCreate))
		assert.False(t, containsTextEvent(out, 159632, trace.EvProcStart))
		assert.True(t, containsTextEvent(out, 1001123792, trace.EvGoStart))
		assert.True(t, containsTextEvent(out, 1001309920, trace.EvGoSched))
	})

	t.Run("G Filter", func(t *testing.T) {
		f := DefaultEventFilter()
		f.G = 1
		out := events(t, inTrace, f)
		assert.True(t, containsTextEvent(out, 0, trace.EvGoCreate))
		assert.False(t, containsTextEvent(out, 159632, trace.EvProcStart))
		assert.True(t, containsTextEvent(out, 1001121216, trace.EvGoUnblock))
		assert.True(t, containsTextEvent(out, 1001123792, trace.EvGoStart))
		assert.True(t, containsTextEvent(out, 1001309920, trace.EvGoSched))
	})

	t.Run("Stack Filter", func(t *testing.T) {
		f := DefaultEventFilter()
		f.StackIDs = []uint32{8, 11}
		out := events(t, inTrace, f)

		assert.False(t, containsTextEvent(out, 0, trace.EvGoCreate))
		assert.True(t, containsTextEvent(out, 21920, trace.EvGoCreate)) // stk=9 stack=8
		assert.True(t, containsTextEvent(out, 23168, trace.EvGoCreate)) // stk=11 stack=10
		assert.False(t, containsTextEvent(out, 29536, trace.EvGoCreate))
	})

	t.Run("Verbose", func(t *testing.T) {
		f := DefaultEventFilter()
		f.Verbose = true
		out := events(t, inTrace, f)
		require.True(t, strings.Contains(out, "stack 8:"))
		require.True(t, strings.Contains(out, "runtime/trace.Start.func1()"))
	})
}

func TestStacks(t *testing.T) {
	inTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "1.19", "trace.bin"))
	require.NoError(t, err)

	t.Run("Default Filter", func(t *testing.T) {
		out := stacks(t, inTrace, DefaultStackFilter())
		require.True(t, strings.Contains(out, "stack 1:"))
		require.True(t, strings.Contains(out, "stack 16:"))
	})

	t.Run("IDs Filter", func(t *testing.T) {
		out := stacks(t, inTrace, StackFilter{StackIDs: []uint32{8, 15}})
		require.False(t, strings.Contains(out, "stack 1:"))

		require.True(t, strings.Contains(out, "stack 8:"))
		require.True(t, strings.Contains(out, "runtime/trace.Start.func1()"))

		require.True(t, strings.Contains(out, "stack 15:"))
		require.True(t, strings.Contains(out, "runtime.asyncPreempt()"))
		require.True(t, strings.Contains(out, "main.main.func1()"))

		require.False(t, strings.Contains(out, "stack 16:"))
	})
}

func events(t *testing.T, in []byte, filter EventFilter) string {
	t.Helper()
	var out bytes.Buffer
	err := Events(bytes.NewReader(in), &out, filter)
	require.NoError(t, err)
	return out.String()
}

func stacks(t *testing.T, in []byte, filter StackFilter) string {
	t.Helper()
	var out bytes.Buffer
	err := Stacks(bytes.NewReader(in), &out, filter)
	require.NoError(t, err)
	return out.String()
}

func containsTextEvent(s string, ts int64, typ byte) bool {
	desc := &trace.EventDescriptions[typ]
	pattern := fmt.Sprintf("(?m)^%d %s.+", ts, desc.Name)
	ok, err := regexp.MatchString(pattern, s)
	return ok && err == nil
}
