package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"

	"github.com/felixge/traceutils/pkg/encoding"
)

func main() {
	cmd := Cmd{}
	flag.StringVar(&cmd.CPUProfile, "cpuprofile", "", "write cpu profile to file")
	flag.Parse()
	cmd.File = flag.Arg(0)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

type Cmd struct {
	File       string
	CPUProfile string
}

func (c *Cmd) Run() error {
	cpuFile, err := os.Create(c.CPUProfile)
	if err != nil {
		return err
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return err
	}
	defer pprof.StopCPUProfile()

	file, err := os.Open(c.File)
	if err != nil {
		return err
	}
	defer file.Close()

	p, err := encoding.NewDecoder(file)
	if err != nil {
		return err
	}

	var n int
	for {
		var e encoding.Event
		if err := p.Decode(&e); err == io.EOF {
			fmt.Printf("%d events parsed\n", n)
			return nil
		} else if err != nil {
			return err
		}

		if e.Type == encoding.EventString {
			fmt.Printf("%s\n", e.Str)
		}
		n++
	}
}
