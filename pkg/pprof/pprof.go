package pprof

import (
	"fmt"
	"io"
	"time"

	"github.com/google/pprof/profile"
	"honnef.co/go/gotraceui/trace"
)

type Options struct {
}

func Convert(r io.Reader, w io.Writer, opt Options) error {
	t, err := trace.Parse(r, nil)
	if err != nil {
		return err
	}

	p := &profile.Profile{
		SampleType:        []*profile.ValueType{{Type: "wall-time", Unit: "nanoseconds"}},
		DefaultSampleType: "wall-time",
		Mapping:           []*profile.Mapping{},
		TimeNanos:         0,
		DurationNanos:     int64(t.Events[len(t.Events)-1].Ts - t.Events[0].Ts),
		// PeriodType: &profile.ValueType{
		// 	Type: "",
		// 	Unit: "",
		// },
		// Period: 0,
	}

	sampleIdx := map[sampleKey]*profile.Sample{}
	locationsIdx := map[uint64][]*profile.Location{}
	locationIdx := map[uint64]*profile.Location{}
	fnIdx := map[funcKey]*profile.Function{}

	pprofSample := func(stack int64, state gSchedState, dt time.Duration) {
		key := sampleKey{string(state), stack}
		sample, ok := sampleIdx[key]
		if !ok {
			locations, ok := locationsIdx[uint64(stack)]
			if !ok {
				pcs := t.Stacks[uint32(stack)]
				for _, pc := range pcs {
					location, ok := locationIdx[pc]
					if !ok {
						frame := t.PCs[pc]
						key := funcKey{Name: frame.Fn, File: frame.File}
						fn, ok := fnIdx[key]
						if !ok {
							fn = &profile.Function{
								ID:       uint64(len(p.Function) + 1),
								Name:     frame.Fn,
								Filename: frame.File,
							}
							p.Function = append(p.Function, fn)
							fnIdx[key] = fn
						}

						location = &profile.Location{
							ID:      uint64(len(p.Location)) + 1,
							Mapping: nil,
							Address: pc,
							Line: []profile.Line{{
								Function: fn,
								Line:     int64(frame.Line),
							}},
						}
						p.Location = append(p.Location, location)
						locationIdx[pc] = location
					}

					// skip runtime.goexit frames from CPU samples
					if !isInternalLocation(location) {
						locations = append(locations, location)
					}
				}
				locationsIdx[uint64(stack)] = locations
			}

			labels := map[string][]string{"state": {string(state)}}
			sample = &profile.Sample{
				Location: locations,
				Value:    []int64{0},
				Label:    labels,
			}
			p.Sample = append(p.Sample, sample)
			sampleIdx[key] = sample
		}

		sample.Value[0] += dt.Nanoseconds()

	}

	var runningTime time.Duration
	transitionState := func(from, to gSchedState, s gState, g uint64, e trace.Event) (gState, error) {
		if s.sched != from {
			return s, fmt.Errorf("g %d: expected state %s, got %s: %s", g, from, s.sched, e.String())
		} else if s.stack == -1 {
			return s, fmt.Errorf("g %d: no stack: %s", g, e.String())
		}

		dt := time.Duration(e.Ts - s.since)
		if s.sched == gSchedRunning {
			runningTime += dt
		} else if s.sched != gSchedInit {
			pprofSample(s.stack, s.sched, dt)
		}

		s.sched = to
		s.since = e.Ts
		return s, nil
	}

	gStates := map[uint64]gState{}
	var cpuSamples int
	for _, e := range t.Events {
		g := e.G
		switch e.Type {
		case trace.EvGoCreate,
			trace.EvGoUnblock,
			trace.EvGoUnblockLocal,
			trace.EvGoSysExit,
			trace.EvGoSysExitLocal:
			g = e.Args[0]
		}

		s := gStates[g]
		if s.sched == "" {
			s.sched = gSchedInit
		}
		switch e.Type {
		// init -> runnable
		case trace.EvGoCreate:
			s, err = transitionState(gSchedInit, gSchedRunnable, s, g, e)
			s.stack = int64(e.Args[1])

		// runnable -> running
		case trace.EvGoStart,
			trace.EvGoStartLocal,
			trace.EvGoStartLabel:
			s, err = transitionState(gSchedRunnable, gSchedRunning, s, g, e)

		// running -> running
		case trace.EvGoSysCall:
			// not calling transitionState here, state doesn't change
			s.stack = int64(e.StkID)

		// running -> runnable
		case trace.EvGoSched, trace.EvGoPreempt:
			s, err = transitionState(gSchedRunning, gSchedRunnable, s, g, e)
			s.stack = int64(e.StkID)

		// running -> waiting
		case trace.EvGoBlockCond,
			trace.EvGoBlockNet,
			trace.EvGoBlockGC,
			trace.EvGoSysBlock,
			trace.EvGoSleep,
			trace.EvGoBlock,
			trace.EvGoBlockRecv,
			trace.EvGoBlockSend,
			trace.EvGoBlockSelect,
			trace.EvGoBlockSync:
			s, err = transitionState(gSchedRunning, gSchedWaiting, s, g, e)
			if e.Type != trace.EvGoSysBlock {
				s.stack = int64(e.StkID)
			}

		// waiting -> runnable
		case trace.EvGoUnblock,
			trace.EvGoUnblockLocal,
			trace.EvGoSysExit,
			trace.EvGoSysExitLocal:
			s, err = transitionState(gSchedWaiting, gSchedRunnable, s, g, e)

		// runnable -> waiting
		case trace.EvGoWaiting,
			trace.EvGoInSyscall:
			s, err = transitionState(gSchedRunnable, gSchedWaiting, s, g, e)

		case trace.EvCPUSample:
			cpuSamples++

		default:
			continue
		}
		if err != nil {
			return err
		}
		gStates[g] = s
	}

	weight := runningTime / time.Duration(cpuSamples)
	for _, e := range t.Events {
		if e.Type != trace.EvCPUSample {
			continue
		}
		pprofSample(int64(e.StkID), gSchedRunning, weight)
	}

	return p.Write(w)
}

func isInternalLocation(loc *profile.Location) bool {
	switch loc.Line[0].Function.Name {
	case "runtime.goexit",
		"runtime.main":
		return true
	}
	return false
}

type sampleKey struct {
	State string
	StkID int64
}

type funcKey struct {
	Name string
	File string
}

type gState struct {
	sched gSchedState
	since trace.Timestamp
	stack int64
}

type gSchedState string

const (
	gSchedInit     gSchedState = "init"
	gSchedRunnable gSchedState = "runnable"
	gSchedWaiting  gSchedState = "waiting"
	gSchedRunning  gSchedState = "running"
)
