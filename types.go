package main

import "time"

type Sheet struct {
	ID      int64
	Name    string
	Active  bool
	Entries []Entry
}

type Entry struct {
	ID        int64
	SheetID   int64
	StartTime time.Time
	EndTime   time.Time
	Note      string
	CreatedAt time.Time
}

type DisplayOptions struct {
	Type string // "day", "week", "month", "year"
}
