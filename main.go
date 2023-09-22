package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/max/github-analytics/config"
)

type GithubEvent struct {
	ID        string `gorm:"primaryKey"`
	Type      string
	Actor     string
	Repo      string    `gorm:"index:idx_repo"`
	CreatedAt time.Time `gorm:"index:idx_created_at"`
}

type RepoWatchCount struct {
	Repo        string
	CurrentRank int
	PrevRank    int
	RankChange  int
	WatchCount  int
}

type Cache struct {
	data          []RepoWatchCount
	lastUpdatedAt time.Time
}

var cache Cache

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open(mysql.Open(cfg.Database.DSN), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	// Run database migrations
	db.AutoMigrate(&GithubEvent{})

	// Create a GitHub client using a personal access token or an OAuth2 token.
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Github.Token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Run the GitHub event collector in the background
	go collectGithubEvents(ctx, db, client)

	// Start a web server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleIndex(w, r, db)
	})
	log.Fatal(http.ListenAndServe(cfg.Addr(), nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	// Check the freshness of cache
	if time.Since(cache.lastUpdatedAt) < time.Hour {
		renderTemplate(w, cache.data)
		return
	}

	// Query the database for all events
	var repoWatchCounts []RepoWatchCount

	query := `
	SELECT
		c.repo,
		c.current_rank,
		p.prev_rank,
		COALESCE(p.prev_rank, 101) - c.current_rank AS rank_change,
		c.watch_count
	FROM (
		SELECT
			ROW_NUMBER() OVER (ORDER BY COUNT(*)
				DESC) AS current_rank,
			repo,
			COUNT(*) AS watch_count
		FROM
			github_events
		WHERE
			created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		GROUP BY
			repo
		LIMIT 100) c
		LEFT JOIN (
			SELECT
				ROW_NUMBER() OVER (ORDER BY COUNT(*)
					DESC) AS prev_rank,
				repo
			FROM
				github_events
			WHERE
				created_at BETWEEN DATE_SUB(NOW(), INTERVAL 48 HOUR)
				AND DATE_SUB(NOW(), INTERVAL 24 HOUR)
			GROUP BY
				repo) p ON c.repo = p.repo
	ORDER BY
		c.watch_count DESC;
	`

	err := db.Raw(query).Scan(&repoWatchCounts).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the HTML template
	renderTemplate(w, repoWatchCounts)

	// Update the cache
	cache.data = repoWatchCounts
	cache.lastUpdatedAt = time.Now()
}

func renderTemplate(w http.ResponseWriter, repoWatchCounts []RepoWatchCount) {
	tmpl, err := template.New("index").Parse(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>GitHub Analytics</title>

			<style>
				body {
					font-family: sans-serif;
				}
			</style>
        </head>
        <body>
            <h1>Top Starred Repos</h1>
			<p><small>Most starred repos in the last 24 hours</small></p>

            <table>
				<thead>
					<tr>
						<th>Current Rank</th>
						<th>Repo</th>
						<th>Previous Rank</th>
						<th>Rank Change</th>
						<th>Watch Count</th>
					</tr>
				</thead>
				<tbody>
					{{range .}}
						<tr>
							<td>{{.CurrentRank}}</td>
							<td><a href="https://github.com/{{.Repo}}">{{.Repo}}</a></td>
							<td>{{.PrevRank}}</td>
							<td>{{.RankChange}}</td>
							<td>{{.WatchCount}}</td>
						</tr>
					{{end}}
				</tbody>
            </table>
        </body>
        </html>
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, repoWatchCounts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
