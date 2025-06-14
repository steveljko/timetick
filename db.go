package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// migration quries
	createSheetsTableSQL = `
  CREATE TABLE IF NOT EXISTS sheets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  active INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  )`

	createEntriesTableSQL = `
  CREATE TABLE IF NOT EXISTS entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  sheet_id INTEGER NOT NULL,
  start_time DATETIME,
  end_time DATETIME,
  note TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (sheet_id) REFERENCES sheets(id)
  )`

	// sheet queries
	createSheetSQL          = `INSERT INTO sheets (name) VALUES (?)`
	getAllSheetsSQL         = `SELECT name FROM sheets`
	getSheetIdByNameSQL     = `SELECT id FROM sheets WHERE name = ?`
	getActiveSheetIdSQL     = `SELECT id FROM sheets WHERE active = 1`
	getSheetsWithEntriesSQL = `
  SELECT s.name, e.start_time, e.end_time, e.note
  FROM sheets s
  JOIN entries e ON e.sheet_id = s.id
  WHERE e.start_time >= ? AND e.start_time <= ? AND e.end_time IS NOT NULL
  ORDER BY s.name
  `
	checkSheetExistsSQL    = `SELECT EXISTS(SELECT 1 FROM sheets WHERE name = ?)`
	activateSheetByNameSQL = `UPDATE sheets SET active = 1 WHERE name = ?`
	deactivateAllSheetsSQL = `UPDATE sheets SET active = 0`

	// entry queries
	createEntrySQL               = `INSERT INTO entries (sheet_id, start_time, note) VALUES (?, ?, ?)`
	createFullEntrySQL           = `INSERT INTO entries(sheet_id, start_time, end_time, note) VALUES (?, ?, ?, ?)`
	getTrackingEntrySQL          = `SELECT id, note FROM entries WHERE end_time IS NULL`
	checkEntryHasNoteSQL         = `SELECT note FROM entries WHERE end_time IS NULL LIMIT 1`
	updateEntryEndTimeAndNoteSQL = `UPDATE entries SET end_time = ?, note = ? WHERE id = ?`
)

type Repo struct {
	db *sql.DB
}

func NewRepo(dbPath string) (*Repo, error) {
	// ensure directory exists
	err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// verify connection with database
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &Repo{db: db}

	// run migrations
	if err := repo.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

func (r *Repo) Close() error {
	return r.db.Close()
}

// runs migrations on initial start
func (r *Repo) runMigrations() error {
	tables := []string{
		createSheetsTableSQL,
		createEntriesTableSQL,
	}

	for _, tableSQL := range tables {
		if _, err := r.db.Exec(tableSQL); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// +---------------------+
// |                     |
// |    Sheet Queries    |
// |                     |
// +---------------------+

// checks if a sheet exists by name
func (r *Repo) CheckSheetExists(name string) bool {
	var exists bool
	err := r.db.QueryRow(checkSheetExistsSQL, name).Scan(&exists)
	if err != nil {
		log.Printf("error checking if sheet exists: %v", err)
		return false
	}
	return exists
}

// creates new sheet
func (r *Repo) CreateSheet(name string) error {
	_, err := r.db.Exec(createSheetSQL, name)
	return err
}

// get all sheets
func (r *Repo) GetAllSheets() ([]string, error) {
	rows, err := r.db.Query(getAllSheetsSQL)
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

// gets sheet id by provided name
func (r *Repo) GetSheetIdByName(name string) (int64, error) {
	var id int64

	err := r.db.QueryRow(getSheetIdByNameSQL, name).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no sheet found with name: %s", name)
		}
		return 0, fmt.Errorf("error getting sheet id: %w", err)
	}

	return id, nil
}

// set active sheet by name
func (r *Repo) SetActiveSheet(name string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// deactivate all sheets
	_, err = tx.Exec(deactivateAllSheetsSQL)
	if err != nil {
		return err
	}

	// activate specified sheet
	_, err = tx.Exec(activateSheetByNameSQL, name)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// get active sheet id
func (r *Repo) GetActiveSheetID() (int64, error) {
	var id int64
	err := r.db.QueryRow(getActiveSheetIdSQL).Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

// get all sheets with their entries
func (r *Repo) GetSheetsWithEntries(startTime, endTime time.Time) ([]Sheet, error) {
	rows, err := r.db.Query(getSheetsWithEntriesSQL, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sheetMap := make(map[string]*Sheet)

	for rows.Next() {
		var sheetName string
		var entry Entry

		if err := rows.Scan(&sheetName, &entry.StartTime, &entry.EndTime, &entry.Note); err != nil {
			return nil, err
		}

		sheet, exists := sheetMap[sheetName]
		if !exists {
			sheet = &Sheet{Name: sheetName}
			sheetMap[sheetName] = sheet
		}

		sheet.Entries = append(sheet.Entries, entry)
	}

	sheets := make([]Sheet, 0, len(sheetMap))
	for _, sheet := range sheetMap {
		sheets = append(sheets, *sheet)
	}

	return sheets, nil
}

// +---------------------+
// |                     |
// |    Entry Queries    |
// |                     |
// +---------------------+
func (r *Repo) CreateEntry(sheetID int64, startTime time.Time, note string) error {
	_, err := r.db.Exec(createEntrySQL, sheetID, startTime, note)
	return err
}

func (r *Repo) HasActiveEntryNote() bool {
	var note string

	err := r.db.QueryRow(checkEntryHasNoteSQL).Scan(&note)

	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		return false
	}

	return note != ""
}

// creates full entry in database (used for importing from telegram bot)
func (r *Repo) CreateFullEntry(sheetName string, startTime time.Time, endTime sql.NullTime, note string) error {
	sheetId, err := r.GetSheetIdByName(sheetName)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(createFullEntrySQL, sheetId, startTime, endTime, note)
	return err
}

func (r *Repo) UpdateEntry(endTime time.Time, note string) error {
	var entryID int64
	var existingNote string

	err := r.db.QueryRow(getTrackingEntrySQL).Scan(&entryID, &existingNote)
	if err != nil {
		return fmt.Errorf("error finding entry without end time: %w", err)
	}

	// use provided note param if the existing note is empty
	updateNote := existingNote
	if existingNote == "" && note != "" {
		updateNote = note
	}

	_, err = r.db.Exec(updateEntryEndTimeAndNoteSQL, endTime, updateNote, entryID)
	if err != nil {
		return fmt.Errorf("error while updating end time to entry: %w", err)
	}

	return nil
}
