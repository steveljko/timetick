package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	pkgterm "github.com/pkg/term"
	xterm "golang.org/x/term"
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

// +--------------+
// |              |
// |    Select    |
// |              |
// +--------------+
// This code is adapted from https://github.com/Nexidian/gocliselect
// The only differences are that it has no colors and supports a multiline prompt

// Control sequences for terminal
const (
	HideCursor     = "\033[?25l"
	ShowCursor     = "\033[?25h"
	ClearLine      = "\033[2K"
	CursorUpFormat = "\033[%dA"
)

// Key codes
const (
	KeyEnter  = 13
	KeyEscape = 27
	KeyUp     = 65
	KeyDown   = 66
)

// NavigationKeys maps the byte values of navigation keys
var NavigationKeys = map[byte]bool{
	KeyUp:   true,
	KeyDown: true,
}

type Menu struct {
	Prompt       string
	CursorPos    int
	ScrollOffset int
	MenuItems    []*MenuItem
}

type MenuItem struct {
	Text    string
	ID      string
	SubMenu *Menu
}

func NewMenu(prompt string) *Menu {
	return &Menu{
		Prompt:    prompt,
		MenuItems: make([]*MenuItem, 0),
	}
}

// AddItem will add a new menu option to the menu list
func (m *Menu) AddItem(option string, id string) *Menu {
	menuItem := &MenuItem{
		Text: fmt.Sprintf("%s", option),
		ID:   id,
	}

	m.MenuItems = append(m.MenuItems, menuItem)
	return m
}

// renderMenuItems prints the menu item list.
// Setting redraw to true will re-render the options list with updated current selection.
func (m *Menu) renderMenuItems(redraw bool) {
	_, height, err := xterm.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		fmt.Println("Error getting terminal size:", err)
		return
	}
	termHeight := height - 3 // Space for prompt and cursor movement
	menuSize := len(m.MenuItems)

	// Ensure scroll offset follows cursor movement
	if m.CursorPos < m.ScrollOffset {
		m.ScrollOffset = m.CursorPos
	} else if m.CursorPos >= m.ScrollOffset+termHeight {
		m.ScrollOffset = m.CursorPos - termHeight + 1
	}

	if redraw {
		// Move the cursor up n lines where n is the number of visible options
		fmt.Printf(CursorUpFormat, min(menuSize, termHeight))
	}

	// Render only visible menu items
	for i := m.ScrollOffset; i < min(m.ScrollOffset+termHeight, menuSize); i++ {
		menuItem := m.MenuItems[i]
		cursor := "  "

		fmt.Print(ClearLine)

		if i == m.CursorPos {
			cursor = "> "
		}

		fmt.Printf("\r%s%s\n", cursor, menuItem.Text)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Display will display the current menu options and awaits user selection
// It returns the user's selected choice as a string
func (m *Menu) Display() (string, error) {
	defer func() {
		// Show cursor again.
		fmt.Printf(ShowCursor)
	}()

	if len(m.MenuItems) == 0 {
		return "", errors.New("menu has no items to display")
	}

	// Print the multiline prompt
	lines := strings.Split(m.Prompt, "\n")
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println() // Extra line after prompt

	m.renderMenuItems(false)

	fmt.Printf(HideCursor)

	// Channel to signal interrupt
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interruptChan:
			return "", nil
		default:
			keyCode := getInput()
			switch keyCode {
			case KeyEscape:
				return "", nil
			case KeyEnter:
				menuItem := m.MenuItems[m.CursorPos]
				fmt.Println("\r")
				// Convert ID to string
				return fmt.Sprintf("%v", menuItem.ID), nil
			case KeyUp:
				m.CursorPos = (m.CursorPos + len(m.MenuItems) - 1) % len(m.MenuItems)
				m.renderMenuItems(true)
			case KeyDown:
				m.CursorPos = (m.CursorPos + 1) % len(m.MenuItems)
				m.renderMenuItems(true)
			}
		}
	}
}

// getInput will read raw input from the terminal
// It returns the raw ASCII value inputted
func getInput() byte {
	t, err := pkgterm.Open("/dev/tty")
	if err != nil {
		return 0
	}
	defer t.Close()

	err = pkgterm.RawMode(t)
	if err != nil {
		return 0
	}
	defer t.Restore() // Restore terminal mode

	readBytes := make([]byte, 3)
	read, err := t.Read(readBytes)
	if err != nil {
		return 0
	}

	// Arrow keys are prefixed with the ANSI escape code
	// For example the up arrow key is '<esc>[A' while the down is '<esc>[B'
	if read == 3 && readBytes[0] == KeyEscape && readBytes[1] == '[' {
		if _, ok := NavigationKeys[readBytes[2]]; ok {
			return readBytes[2]
		}
	} else if read >= 1 {
		return readBytes[0]
	}

	return 0
}
