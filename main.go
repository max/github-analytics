package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v40/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GithubEvent struct {
	ID        string `gorm:"primaryKey"`
	Type      string
	Actor     string
	Repo      string
	CreatedAt time.Time
}

func main() {
	// Pick up environment variables from .env
	if err := godotenv.Load(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Println("No .env file found")
		} else {
			log.Fatalf("Error loading .env file: %v\n", err)
		}
	}

	db, err := gorm.Open(mysql.Open(os.Getenv("DSN")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Run database migrations
	db.AutoMigrate(&GithubEvent{})

	// Create a GitHub client using a personal access token or an OAuth2 token.
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Run the GitHub event collector in the background
	go collectGithubEvents(ctx, db, client)

	// Start a web server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, world!")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

}

func collectGithubEvents(ctx context.Context, db *gorm.DB, client *github.Client) {
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

			githubEvent := GithubEvent{
				ID:        event.GetID(),
				Type:      event.GetType(),
				Actor:     event.Actor.GetLogin(),
				Repo:      event.Repo.GetName(),
				CreatedAt: event.GetCreatedAt(),
			}

			result := db.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(&githubEvent)

			if result.Error != nil {
				fmt.Printf("Error inserting event into database: %v\n", result.Error)
				continue
			}

			if result.RowsAffected > 0 {
				fmt.Printf("Inserted event %s\n", event.GetID())
			}
		}
	}
}
