package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"

	"github.com/felixge/traceutils/pkg/breakdown"
	"github.com/olekukonko/tablewriter"
)

type BreakdownFlavor string

const (
	BreakdownCSV   BreakdownFlavor = "csv"
	BreakdownBytes BreakdownFlavor = "size"
	BreakdownCount BreakdownFlavor = "count"
)

func BreakdownCommand(flavor BreakdownFlavor, args []string) error {
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

	// Break down the trace file by event type
	bd, err := breakdown.ByEventType(inFile)
	if err != nil {
		return err
	}

	totalBytes := int64(0)
	totalCount := int64(0)
	summaries := make([]breakdown.EventTypeSummary, 0, len(bd))
	for _, ets := range bd {
		summaries = append(summaries, ets)
		totalBytes += ets.Bytes
		totalCount += ets.Count
	}

	var header []string
	var rows [][]string
	var footer []string
	switch flavor {
	case BreakdownCSV:
		header = []string{"Event Type", "Count", "Bytes"}
		cw := csv.NewWriter(os.Stdout)
		cw.Write(header)
		for _, ets := range summaries {
			cw.Write([]string{
				ets.EventType.String(),
				fmt.Sprintf("%d", ets.Count),
				fmt.Sprintf("%d", ets.Bytes),
			})
		}
		cw.Flush()
		return cw.Error()
	case BreakdownCount:
		header = []string{"Event Type", "Count", "%"}
		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].Count > summaries[j].Count
		})
		for _, ets := range summaries {
			rows = append(rows, []string{
				ets.EventType.String(),
				fmt.Sprintf("%d", ets.Count),
				fmt.Sprintf("%.2f%%", float64(ets.Count)/float64(totalCount)*100),
			})
		}
		footer = []string{"Total", fmt.Sprintf("%d", totalCount), "100.00%"}
	case BreakdownBytes:
		header = []string{"Event Type", "Bytes", "%"}
		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].Bytes > summaries[j].Bytes
		})
		for _, ets := range summaries {
			rows = append(rows, []string{
				ets.EventType.String(),
				humanBytes(ets.Bytes),
				fmt.Sprintf("%.2f%%", float64(ets.Bytes)/float64(totalBytes)*100),
			})
		}
		footer = []string{"Total", humanBytes(totalBytes), "100.00%"}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.AppendBulk(rows)
	table.SetFooter(footer)
	table.Render()
	return nil
}

// humanBytes converts the given byte value to a human readable string.
func humanBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "kMGTPE"[exp])
}
