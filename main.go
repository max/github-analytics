package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
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
	db, err := sql.Open("sqlite3", strings.Replace(os.Getenv("DATABASE_URL"), "sqlite:", "file:", 1))
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

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
			continue
		}

		// Print the event information.
		for _, event := range events {
			if event.GetType() != "WatchEvent" {
				continue
			}

			result, err := db.Exec(`INSERT OR IGNORE INTO events (
				id, type, actor, repo, payload, org, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`, event.GetID(), event.GetType(), event.Actor.GetLogin(), event.Repo.GetName(), event.GetRawPayload(), event.Org.GetName(), event.GetCreatedAt())
			if err != nil {
				fmt.Printf("Error inserting event into database: %v\n", err)
				continue
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				fmt.Printf("Error getting rows affected: %v\n", err)
				continue
			}

			if rowsAffected > 0 {
				fmt.Printf("Inserted event %s\n", event.GetID())
			}
		}
	}
}
