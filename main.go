package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
  db *sql.DB
  sheetName string
)

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
    sheetName = args[0]

    if checkSheetExists(sheetName) {
      fmt.Printf("Changed sheet to: %s\n", sheetName)
    } else {
      createSheet(sheetName)
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

func init() {
	dbPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "timetick", "database.db")

	err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
    log.Fatal("Failed to open database: %v", err)
	}

  if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

  if err := runMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
  }

	rootCmd.AddCommand(sheetCmd)
}

// Run database migrations
func runMigrations() error {
  tables := []string {
    `CREATE TABLE IF NOT EXISTS sheets (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL,
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
