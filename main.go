package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var db *sql.DB

type Entry struct {
	StartTime time.Time
	EndTime   time.Time
	Note      string
}

type Sheet struct {
	Name    string
	Entries []Entry
}

var rootCmd = &cobra.Command{
	Use:   "timetick",
	Short: "X",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to the Time Tracking App")
	},
}

var sheetCmd = &cobra.Command{
	Use:   "sheet [sheetname]",
	Short: "Change tracking sheet",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		sheets, err := getAllSheets()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return sheets, cobra.ShellCompDirectiveNoFileComp
	},
	Run: func(cmd *cobra.Command, args []string) {
		sheetName := args[0]

		if checkSheetExists(sheetName) {
			setActiveSheet(sheetName)
			fmt.Printf("Changed sheet to: %s\n", sheetName)
		} else {
			createSheet(sheetName)
			setActiveSheet(sheetName)
			fmt.Printf("Created and changed sheet to: %s\n", sheetName)
		}
	},
}

func checkSheetExists(name string) bool {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM sheets WHERE name = ?)`
	err := db.QueryRow(query, name).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}
	return exists
}

func createSheet(name string) {
	_, err := db.Exec("INSERT INTO sheets (name) VALUES (?)", name)
	if err != nil {
		log.Fatal(err)
	}
}

func getAllSheets() ([]string, error) {
	rows, err := db.Query("SELECT name FROM sheets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheets []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		sheets = append(sheets, name)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sheets, nil
}

// set active sheet by name
func setActiveSheet(name string) {
	// deactivate all sheets
	_, err := db.Exec("UPDATE sheets SET active = 0")
	if err != nil {
		log.Fatal(err)
	}

	// activate with name
	_, err = db.Exec("UPDATE sheets SET active = 1 WHERE name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
}

// get active sheet id
func getActiveSheetId() (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM sheets WHERE active = 1").Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func getSheetsWithEntries(displayType string) ([]Sheet, error) {
	now := time.Now()
	var startTime, endTime time.Time

	switch displayType {
	case "day":
		startTime = now.Truncate(24 * time.Hour)
		endTime = startTime.Add(24 * time.Hour)
	case "week":
		startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
		startTime = startOfWeek
		endTime = startOfWeek.AddDate(0, 0, 7)
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 1, 0)
	case "year":
		startTime = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(1, 0, 0)
	default:
		return nil, fmt.Errorf("invalid display mode: %s", displayType)
	}

	rows, err := db.Query(`
    SELECT DISTINCT sheets.name
    FROM sheets
    JOIN entries ON entries.sheet_id = sheets.id
    WHERE entries.created_at >= ? AND entries.created_at < ?
    `, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheets []Sheet

	for rows.Next() {
		var sheet Sheet
		if err := rows.Scan(&sheet.Name); err != nil {
			return nil, err
		}

		entriesRows, err := db.Query(`
      SELECT start_time, end_time, note
      FROM entries
      JOIN sheets ON entries.sheet_id = sheets.id
      WHERE sheets.name = ? AND end_time IS NOT NULL
      AND start_time >= ? AND start_time <= ?
      `, sheet.Name, startTime, endTime)
		if err != nil {
			return nil, err
		}
		defer entriesRows.Close()

		var entries []Entry
		for entriesRows.Next() {
			var entry Entry
			if err := entriesRows.Scan(&entry.StartTime, &entry.EndTime, &entry.Note); err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
		sheet.Entries = entries
		sheets = append(sheets, sheet)
	}

	return sheets, nil
}

var startCmd = &cobra.Command{
	Use:   "start [note]",
	Short: "Start tracking time",
	Run: func(cmd *cobra.Command, args []string) {
		var note string
		if len(args) > 0 {
			note = args[0]
		}

		id, _ := getActiveSheetId()
		startTime := time.Now()

		_, err := db.Exec("INSERT INTO entries (sheet_id, start_time, note) VALUES (?, ?, ?)", id, startTime, note)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Started tracking time...")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop tracking time",
	Run: func(cmd *cobra.Command, args []string) {
		// Disable stop when there is no entry started.
		endtime := time.Now()

		var entryId int64
		var note string
		err := db.QueryRow("SELECT id, note FROM entries WHERE end_time IS NULL").Scan(&entryId, &note)
		if err != nil {
			log.Fatalf("Error finding entry without end time: %v", err)
		}

		if note != "" {
			_, err = db.Exec("UPDATE entries SET end_time = ? WHERE id = ?", endtime, entryId)
			if err != nil {
				log.Fatalf("Error while updating end time to entry: %v", err)
			}
		} else {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter a note (press Enter to skip): ")
			note, _ := reader.ReadString('\n')
			note = strings.TrimSpace(note)

			_, err = db.Exec("UPDATE entries SET end_time = ?, note = ? WHERE id = ?", endtime, note, entryId)
			if err != nil {
				log.Fatalf("Error while updating end time to entry: %v", err)
			}
		}

		fmt.Println("Tracking stopped!")
	},
}

func printTable(headers []string, rows [][]string, footers []string) {
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

	// Print header
	for i, header := range headers {
		fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], header)
	}
	fmt.Fprintln(os.Stdout)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], cell)
		}
		fmt.Fprintln(os.Stdout)
	}

	// Print footers
	for i, footer := range footers {
		if footer != "" {
			fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], footer)
		} else {
			// Print empty space for skipped footer
			fmt.Fprintf(os.Stdout, "%-*s\t", colWidths[i], "")
		}
	}
	fmt.Fprintln(os.Stdout)
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

var displayCmd = &cobra.Command{
	Use:   "display [type]",
	Short: "Asd",
	Run: func(cmd *cobra.Command, args []string) {
		displayType := "day"
		if len(args) > 0 {
			displayType = args[0]
		}

		sheets, err := getSheetsWithEntries(displayType)
		if err != nil {
			log.Fatalf("Error fetching sheets with entries: %v", err)
		}

		for _, sheet := range sheets {
			fmt.Printf("Timesheet: %s\n", sheet.Name)
			headers := []string{"Day", "Start", "End", "Duration", "Notes"}

			var rows [][]string
			totalDuration := time.Duration(0)

			var lastDay string
			for _, entry := range sheet.Entries {
				day := entry.StartTime.Format("Jan 02, 2006")
				startTime := entry.StartTime.Format("15:04:05")
				endTime := entry.EndTime.Format("15:04:05")
				duration := entry.EndTime.Sub(entry.StartTime)
				totalDuration += duration

				if day != lastDay {
					row := []string{
						day,
						startTime,
						endTime,
						formatDuration(duration),
						entry.Note,
					}

					rows = append(rows, row)
					lastDay = day
				} else {
					row := []string{
						"",
						startTime,
						endTime,
						formatDuration(duration),
						entry.Note,
					}

					rows = append(rows, row)
				}
			}

			footers := []string{"", "", "Total:", formatDuration(totalDuration), ""}
			printTable(headers, rows, footers)
		}
	},
}

func init() {
	dbPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "timetick", "database.db")

	err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	if err := runMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	rootCmd.AddCommand(sheetCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(displayCmd)
}

// Run database migrations
func runMigrations() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS sheets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`,
		`CREATE TABLE IF NOT EXISTS entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sheet_id INTEGER NOT NULL,
    start_time DATETIME,
    end_time DATETIME,
    note TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (sheet_id) REFERENCES sheets(id)
    );`,
	}

	for _, tableSQL := range tables {
		if _, err := db.Exec(tableSQL); err != nil {
			return fmt.Errorf("Failed to create table: %w", err)
		}
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
