package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	apiManager "github.com/TheKrainBow/go-api"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type dbUser struct {
	ID      string `db:"id"`
	FtLogin string `db:"ft_login"`
	Email   string `db:"email"`
}

type user42EmailResponse struct {
	Email string `json:"email"`
}

func main() {
	overwrite := flag.Bool("overwrite", false, "refresh email even when users.email is already set")
	delay := flag.Duration("delay", 250*time.Millisecond, "delay between 42 API requests")
	flag.Parse()

	_ = godotenv.Load("../../.env")

	db, err := sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	client, err := apiManager.NewAPIClient("42", apiManager.APIClientInput{
		AuthType:     apiManager.AuthTypeClientCredentials,
		TokenURL:     "https://api.intra.42.fr/oauth/token",
		Endpoint:     "https://api.intra.42.fr/v2",
		TestPath:     "/campus/41",
		ClientID:     os.Getenv("FT_CLIENT_ID"),
		ClientSecret: os.Getenv("FT_CLIENT_SECRET"),
		Scope:        "public",
	})
	if err != nil {
		log.Fatalf("init 42 API client: %v", err)
	}
	if err := client.TestConnection(); err != nil {
		log.Fatalf("42 API connection failed: %v", err)
	}

	var users []dbUser
	err = db.Select(&users, `
		SELECT id, ft_login, COALESCE(email, '') AS email
		FROM users
		ORDER BY id ASC
	`)
	if err != nil {
		log.Fatalf("list users: %v", err)
	}

	var updated int
	var skipped int
	var failed int
	for i, user := range users {
		if strings.TrimSpace(user.Email) != "" && !*overwrite {
			skipped++
			continue
		}

		email, err := fetch42Email(user.FtLogin)
		if err != nil {
			failed++
			log.Printf("[%d/%d] %s: fetch failed: %v", i+1, len(users), user.FtLogin, err)
		} else {
			email = strings.TrimSpace(email)
			if email == "" {
				skipped++
				log.Printf("[%d/%d] %s: no email returned by 42", i+1, len(users), user.FtLogin)
			} else if _, err := db.Exec(`UPDATE users SET email = NULLIF($1, '') WHERE id = $2`, email, user.ID); err != nil {
				failed++
				log.Printf("[%d/%d] %s: update failed: %v", i+1, len(users), user.FtLogin, err)
			} else {
				updated++
				log.Printf("[%d/%d] %s: updated email", i+1, len(users), user.FtLogin)
			}
		}

		if *delay > 0 && i < len(users)-1 {
			time.Sleep(*delay)
		}
	}

	fmt.Printf("done: updated=%d skipped=%d failed=%d total=%d\n", updated, skipped, failed, len(users))
	if failed > 0 {
		os.Exit(1)
	}
}

func fetch42Email(login string) (string, error) {
	path := fmt.Sprintf("/users/%s", url.PathEscape(login))
	resp, err := apiManager.GetClient("42").Get(path)
	if err != nil {
		return "", fmt.Errorf("42 API GET %s: %w", path, err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf("42 API GET %s: status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out user42EmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("42 API decode %s: %w", path, err)
	}
	return out.Email, nil
}
