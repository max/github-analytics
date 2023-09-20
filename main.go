package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v40/github"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

func main() {
	// Pick up environment variables from .env
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Create the events table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS events (
			id TEXT,
			type TEXT,
			actor TEXT,
			repo TEXT,
			payload TEXT,
			org TEXT,
			created_at TEXT
	)`)
	if err != nil {
		log.Fatalf("Error creating events table: %v", err)
	}

	// Create a GitHub client using a personal access token or an OAuth2 token.
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// List all public events on GitHub.
		events, _, err := client.Activity.ListEvents(ctx, &github.ListOptions{PerPage: 100})
		if err != nil {
			fmt.Printf("Error fetching GitHub events: %v\n", err)
			os.Exit(1)
		}

		// Print the event information.
		for _, event := range events {
			if event.GetType() != "WatchEvent" {
				continue
			}

			_, err := db.Exec(`INSERT INTO events (
				id, type, actor, repo, payload, org, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`, event.GetID(), event.GetType(), event.Actor.GetLogin(), event.Repo.GetName(), event.GetRawPayload(), event.Org.GetName(), event.GetCreatedAt())
			if err != nil {
				fmt.Printf("Error inserting event into database: %v\n", err)
				continue
			}
		}
	}
}
