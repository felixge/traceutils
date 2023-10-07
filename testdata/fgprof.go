//go:build ignore

// This code is taken from https://github.com/felixge/fgprof

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"net/http/httptest"
	_ "net/http/pprof"
)

const (
	sleepTime   = 10 * time.Millisecond
	cpuTime     = 30 * time.Millisecond
	networkTime = 60 * time.Millisecond
)

// sleepURL is the url for the sleep server used by slowNetworkRequest. It's
// a global variable to keep the cute simplicity of main's loop.
var sleepURL string

func main() {
	// Start a sleep server to help with simulating slow network requests.
	var stop func()
	sleepURL, stop = StartSleepServer()
	defer stop()

	// Start the CPU profile.
	if err := pprof.StartCPUProfile(io.Discard); err != nil {
		panic(err)
	}

	// Start the trace.
	if err := trace.Start(os.Stdout); err != nil {
		panic(err)
	}
	time.AfterFunc(3*time.Second, func() {
		trace.Stop()
		os.Exit(0)
	})

	for {
		// Http request to a web service that might be slow.
		slowNetworkRequest()
		// Some heavy CPU computation.
		cpuIntensiveTask()
		// Poorly named function that you don't understand yet.
		weirdFunction()
	}
}

func slowNetworkRequest() {
	res, err := http.Get(sleepURL + "/?sleep=" + networkTime.String())
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		panic(fmt.Sprintf("bad code: %d", res.StatusCode))
	}
}

func cpuIntensiveTask() {
	start := time.Now()
	for time.Since(start) <= cpuTime {
		// Spend some time in a hot loop to be a little more realistic than
		// spending all time in time.Since().
		for i := 0; i < 1000; i++ {
			_ = i
		}
	}
}

func weirdFunction() {
	time.Sleep(sleepTime)
}

// StartSleepServer starts a server that supports a ?sleep parameter to
// simulate slow http responses. It returns the url of that server and a
// function to stop it.
func StartSleepServer() (url string, stop func()) {
	server := httptest.NewServer(http.HandlerFunc(sleepHandler))
	return server.URL, server.Close
}

func sleepHandler(w http.ResponseWriter, r *http.Request) {
	sleep := r.URL.Query().Get("sleep")
	sleepD, err := time.ParseDuration(sleep)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "bad duration: %s: %s\n", sleep, err)
	}
	time.Sleep(sleepD)
	fmt.Fprintf(w, "slept for: %s\n", sleepD)
}
