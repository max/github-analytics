package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/go-github/v40/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func main() {
	// Pick up environment variables from .env
	if err := godotenv.Load(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Println("No .env file found")
		} else {
			log.Fatalf("Error loading .env file: %v\n", err)
		}
	}

	// Database connection string
	dbUrl, _ := url.Parse(os.Getenv("DATABASE_URL"))
	username := dbUrl.User.Username()
	password, _ := dbUrl.User.Password()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, dbUrl.Hostname(), dbUrl.Port(), dbUrl.Path[1:])

	// Run database migrations
	dbm := dbmate.New(dbUrl)

	err := dbm.CreateAndMigrate()
	if err != nil {
		log.Fatalf("Error migrating database: %v", err)
	}

	// Open MySQL database connection
	db, err := sql.Open("mysql", dsn)
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

			result, err := db.Exec(`INSERT IGNORE INTO events (
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
