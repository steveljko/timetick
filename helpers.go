package main

import (
	"fmt"
	"os"
	"time"
)

func PrintTable(headers []string, rows [][]string, footers []string) {
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// print header
	for i, header := range headers {
		fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], header)
	}
	fmt.Fprintln(os.Stdout)

	// print rows
	for _, row := range rows {
		for i, cell := range row {
			fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], cell)
		}
		fmt.Fprintln(os.Stdout)
	}

	// print footer
	for i, footer := range footers {
		if footer != "" {
			fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], footer)
		} else {
			// print empty space for skipped footer
			fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], "")
		}
	}
	fmt.Fprintln(os.Stdout)
}

func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}
