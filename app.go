package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
)

func (a *App) ChangeSheet(name string) error {
	if a.repo.CheckSheetExists(name) {
		if err := a.repo.SetActiveSheet(name); err != nil {
			return err
		}

		fmt.Printf("Changed sheet to: %s\n", name)
	} else {
		if err := a.repo.CreateSheet(name); err != nil {
			return err
		}
		if err := a.repo.SetActiveSheet(name); err != nil {
			return err
		}

		fmt.Printf("Created and changed sheet to: %s\n", name)
	}

	return nil
}

func (a *App) StartTracking(note string) error {
	id, err := a.repo.GetActiveSheetID()
	if err != nil {
		return err
	}
	if id == 0 {
		return fmt.Errorf("No active sheet selected, use 'sheet' command to select or create new one")
	}

	startTime := time.Now()
	if err := a.repo.CreateEntry(id, startTime, note); err != nil {
		return err
	}

	fmt.Println("Started tracking time...")
	return nil
}

func (a *App) StopTracking(note string) error {
	endTime := time.Now()

	if a.repo.HasActiveEntryNote() == false && note == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter a note (press Enter to skip): ")
		inputNote, _ := reader.ReadString('\n')
		note = strings.TrimSpace(inputNote)
	}

	err := a.repo.UpdateEntry(endTime, note)
	if err != nil {
		return err
	}

	fmt.Println("Tracking stopped!")
	return nil
}

func (a *App) Display(displayType string) error {
	now := time.Now()
	var startTime, endTime time.Time

	switch displayType {
	case "day":
		startTime = now.Truncate(24 * time.Hour)
		endTime = startTime.Add(24 * time.Hour)
	case "week":
		startOfWeek := now.AddDate(0, 0, -int(now.Weekday())+1)
		if now.Weekday() == time.Sunday {
			startOfWeek = startOfWeek.AddDate(0, 0, -6)
		}
		startTime = startOfWeek
		endTime = startOfWeek.AddDate(0, 0, 7)
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(0, 1, 0)
	case "year":
		startTime = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endTime = startTime.AddDate(1, 0, 0)
	default:
		return fmt.Errorf("Invalid display mode: %s", displayType)
	}

	sheets, err := a.repo.GetSheetsWithEntries(startTime, endTime)
	if err != nil {
		fmt.Errorf("a %v", err)
	}
	for _, sheet := range sheets {
		fmt.Printf("Sheet - %s\n", sheet.Name)

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
					FormatDuration(duration),
					entry.Note,
				}

				rows = append(rows, row)
				lastDay = day
			} else {
				row := []string{
					"",
					startTime,
					endTime,
					FormatDuration(duration),
					entry.Note,
				}

				rows = append(rows, row)
			}
		}

		footers := []string{"", "", "Total:", FormatDuration(totalDuration), ""}
		PrintTable(headers, rows, footers)

		fmt.Println()
		fmt.Println()
	}

	return nil
}

func (a *App) Import(url string) (string, error) {
	apiClient := NewAPIClient(url)

	var IDs []int64

	unimportedEntries, err := apiClient.GetUnimportedEntries()
	if err != nil {
		return "", err
	}

	for _, entry := range unimportedEntries {
		var endTime sql.NullTime
		if entry.EndTime.Valid {
			endTime = sql.NullTime{
				Time:  entry.EndTime.Time,
				Valid: true,
			}
		} else {
			endTime = sql.NullTime{Valid: false}
		}

		err := a.repo.CreateFullEntry(entry.StartTime, endTime, entry.Note)
		if err != nil {
			return "", err
		}

		IDs = append(IDs, int64(entry.ID))
	}

	msg, err := apiClient.MarkEntriesAsImported(IDs)

	return msg, nil
}
