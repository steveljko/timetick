package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// +-------------+
// |             |
// |    Table    |
// |             |
// +-------------+
func PrintTable(headers []string, rows [][]string, footers []string) {
	colWidths := make([]int, len(headers))

	// calc initial column width using header width
	for i, header := range headers {
		colWidths[i] = len(header)
	}
	// if cell is grather than column width, make it cell width
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

// converts duration value into a formatted string
// in the format "hours:minutes:seconds" (H:MM:SS).
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// clears terminal screen
func clearScreen() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	default: // for unix based systems
		cmd = exec.Command("clear")
	}

	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
