package main

import (
	"bufio"
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
