package main

import (
	"database/sql"
  "time"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
  Run: func(cmd *cobra.Command, args []string) {
    sheetName := args[0]

    if checkSheetExists(sheetName) {
      setActiveSheet(sheetName);
      fmt.Printf("Changed sheet to: %s\n", sheetName)
    } else {
      createSheet(sheetName)
      setActiveSheet(sheetName);
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

func getSheetsWithEntries() ([]Sheet, error) {
  sheetsQuery := `SELECT name FROM sheets`
  rows, err := db.Query(sheetsQuery)
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

    entries, err := getEntriesForSheet(sheet.Name)
    if err != nil {
      return nil, err
    }
    sheet.Entries = entries
    sheets = append(sheets, sheet)
  }

  return sheets, nil
}

func getEntriesForSheet(sheetName string) ([]Entry, error) {
  query := `
    SELECT start_time, end_time, note
    FROM entries
    JOIN sheets ON entries.sheet_id = sheets.id
    WHERE sheets.name = ? AND end_time IS NOT NULL
  `

  rows, err := db.Query(query, sheetName)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  var entries []Entry
  for rows.Next() {
    var entry Entry
    if err := rows.Scan(&entry.StartTime, &entry.EndTime, &entry.Note); err != nil {
      return nil, err
    }
    entries = append(entries, entry)
  }

  return entries, nil
}

var startCmd = &cobra.Command{
  Use:   "start [note]",
  Short: "Start tracking time",
  Args:  cobra.ExactArgs(1),
  Run: func(cmd *cobra.Command, args []string) {
    note := args[0]
    startTime := time.Now()

    id, _ := getActiveSheetId()

    _, err := db.Exec("INSERT INTO entries (sheet_id, start_time, note) VALUES (?, ?, ?)", id, startTime, note)
    if err != nil {
      log.Fatal(err)
    }

    fmt.Printf("Started tracking time with note: %s at %s\n", note, startTime)
  },
}

var stopCmd = &cobra.Command{
  Use:   "stop",
  Short: "Stop tracking time",
  Run: func(cmd *cobra.Command, args []string) {
    // Disable stop when there is no entry started.
    endtime := time.Now()

    var entryId int64
    err := db.QueryRow("SELECT id FROM entries WHERE end_time IS NULL").Scan(&entryId)
    if err != nil {
      log.Fatalf("Error finding entry without end time: %v", err)
    }

    _, err = db.Exec("UPDATE entries SET end_time = ? WHERE id = ?", endtime, entryId)
    if err != nil {
      log.Fatalf("Error while updating end time to entry: %v", err)
    }

    fmt.Printf("Stopeed tracking time with entry: %d\n", entryId)
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
  Use:   "display",
  Short: "Asd",
  Run: func(cmd *cobra.Command, args []string) {
    sheets, err := getSheetsWithEntries()
    if err != nil {
      log.Fatalf("Error fetching sheets and entries: %v", err)
    }

    for _, sheet := range sheets {
      fmt.Printf("Timesheet: %s\n", sheet.Name)
      headers := []string{"Day", "Start", "End", "Duration", "Notes"}

      var rows [][]string
      totalDuration := time.Duration(0)

      for _, entry := range sheet.Entries {
        day := entry.StartTime.Format("Jan 02, 2006")
        startTime := entry.StartTime.Format("15:04:05")
        endTime := entry.EndTime.Format("15:04:05")
        duration := entry.EndTime.Sub(entry.StartTime)
        totalDuration += duration

        row := []string{
          day,
          startTime,
          endTime,
          formatDuration(duration),
          entry.Note,
        }

        rows = append(rows, row)
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
  tables := []string {
    `CREATE TABLE IF NOT EXISTS sheets (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL,
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
