package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

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
	fmt.Println("Hello!")
}
