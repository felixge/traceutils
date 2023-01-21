package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/felixge/traceutils/pkg/stw"
	"github.com/olekukonko/tablewriter"
)

func STWCommand(args []string) error {
	// Check the number of arguments
	if len(args) != 1 {
		return fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	// Open the input file
	inFile, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	// Get all the stw events
	events, err := stw.Events(inFile)
	if err != nil {
		return fmt.Errorf("failed to parse events: %w", err)
	}

	// Sort them in descending duration order
	sort.Slice(events, func(i, j int) bool {
		return events[i].Duration() > events[j].Duration()
	})

	// Create table data
	header := []string{"Duration", "Start", "Percentile"}
	var rows [][]string
	for i, e := range events {
		percentile := 100 - float64(i)/float64(len(events))*100
		rows = append(rows, []string{
			e.Duration().String(),
			e.Start.String(),
			fmt.Sprintf("%.2f", percentile),
		})
	}

	// Render the table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.AppendBulk(rows)
	table.Render()

	return nil
}
