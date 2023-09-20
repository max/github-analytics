package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v40/github"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

func main() {
	// Pick up environment variables from .env
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Create a GitHub client using a personal access token or an OAuth2 token.
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	ticker := time.NewTicker(1500 * time.Millisecond)
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

			fmt.Printf("Event ID: %v\n", event.GetID())
			fmt.Printf("Type: %s\n", event.GetType())
			fmt.Printf("Created At: %s\n", event.GetCreatedAt())
			fmt.Printf("Repo: %s\n", event.Repo.GetName())
			fmt.Printf("Actor: %s\n", event.Actor.GetLogin())
			fmt.Println("-------------------------")
		}
	}
}
