package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"

	"github.com/felixge/traceutils/pkg/stw"
	"github.com/olekukonko/tablewriter"
)

type STWFlavor string

const (
	STWCSV STWFlavor = "csv"
	STWTop STWFlavor = "top"
)

func STWCommand(flavor STWFlavor, args []string) error {
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

	switch flavor {
	case STWCSV:
		// Sort them in ascending time order
		sort.Slice(events, func(i, j int) bool {
			return events[i].Start < events[j].Start
		})

		// Write events to stdout in CSV format
		header := []string{"Start (ms)", "Duration (ms)", "Type"}
		cw := csv.NewWriter(os.Stdout)
		cw.Write(header)
		for _, e := range events {
			cw.Write([]string{
				fmt.Sprintf("%f", e.Start.Seconds()*1000),
				fmt.Sprintf("%f", e.Duration().Seconds()*1000),
				string(e.Type),
			})
		}
		cw.Flush()
		return cw.Error()
	case STWTop:
		// Sort them in descending duration order
		sort.Slice(events, func(i, j int) bool {
			return events[i].Duration() > events[j].Duration()
		})

		// Build the table
		header := []string{"Duration", "Start", "Type", "Percentile"}
		var rows [][]string
		for i, e := range events {
			percentile := 100 - float64(i)/float64(len(events))*100
			rows = append(rows, []string{
				e.Duration().String(),
				e.Start.String(),
				string(e.Type),
				fmt.Sprintf("%.2f", percentile),
			})
		}

		// Render the table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(header)
		table.AppendBulk(rows)
		table.Render()
	default:
		return fmt.Errorf("unknown flavor: %s", flavor)
	}
	return nil
}
