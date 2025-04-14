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
}

func main() {
	fmt.Println("Hello!")
}
