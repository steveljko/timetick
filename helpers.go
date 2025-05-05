package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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

// +----------------+
// |                |
// |    Selector    |
// |                |
// +----------------+
type (
	SelectOption struct {
		Value string
		Label string
	}

	TerminalSelector struct {
		Title   string
		Options []SelectOption
	}
)

func NewTerminalSelector(title string, options []SelectOption) *TerminalSelector {
	return &TerminalSelector{
		Title:   title,
		Options: options,
	}
}

// select displays the selection menu and returns the value of the selected option
func (s *TerminalSelector) Select() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	clearScreen()

	// display title and options with index
	fmt.Println(s.Title)
	fmt.Println()

	for i, option := range s.Options {
		fmt.Printf("%d. %s\n", i+1, option.Label)
	}

	// prompt user
	fmt.Println("\nUse number keys (1-" + strconv.Itoa(len(s.Options)) + ") to select an option")
	fmt.Print("\nYour selection: ")

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Print("Please make a selection (1-" + strconv.Itoa(len(s.Options)) + "): ")
			continue
		}

		selection, err := strconv.Atoi(input)
		if err != nil || selection < 1 || selection > len(s.Options) {
			fmt.Print("Invalid selection. Please enter a number (1-" + strconv.Itoa(len(s.Options)) + "): ")
			continue
		}

		clearScreen()

		return s.Options[selection-1].Value, nil
	}
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

// prompts user in terminal to select option and returns the selected value.
func SelectFromOptions(title string, values []string) (string, error) {
	options := make([]SelectOption, len(values))
	for i, value := range values {
		options[i] = SelectOption{
			Value: value,
			Label: value,
		}
	}

	selector := NewTerminalSelector(title, options)
	return selector.Select()
}

// converts duration value into a formatted string
// in the format "hours:minutes:seconds" (H:MM:SS).
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}
