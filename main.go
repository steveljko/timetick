package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type App struct {
	repo *Repo
}

func NewApp(repo *Repo) *App {
	return &App{
		repo: repo,
	}
}

func main() {
	// initilize repo
	dbPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "timetick", "database.db")

	repo, err := NewRepo(dbPath)
	if err != nil {
		fmt.Errorf("failed to create repository: %v", err)
	}
	defer repo.Close()

	// initilize app
	app := NewApp(repo)

	cmd := SetupCommands(app)

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
