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
