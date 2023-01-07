package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cmd := Cmd{}
	flag.Parse()
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

type Cmd struct {
}

func (c *Cmd) Run() error {
	return nil
}
