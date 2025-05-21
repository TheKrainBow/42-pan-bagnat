package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Role struct {
	ID    string `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name  string `json:"name" example:"captain-hook"`
	Color string `json:"color" example:"0xFF00FF"`
}

type Module struct {
	ID            string    `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name          string    `json:"name" example:"captain-hook"`
	Version       string    `json:"version" example:"1.2"`
	Status        string    `json:"status" example:"enabled"`
	URL           string    `json:"url" example:"https://github.com/some-user/some-repo"`
	IconeURL      string    `json:"iconUrl" example:"https://someURL/image.png"`
	LatestVersion string    `json:"lastestVersion" example:"1.7"`
	LateCommits   int       `json:"lateCommits" example:"2"`
	LastUpdate    time.Time `json:"lastUpdate" example:"2025-02-18T15:00:00Z"`
}

type User struct {
	ID        string    `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtLogin   string    `json:"login" example:"heinz"`
	FtID      string    `json:"ftId" example:"1492"`
	FtIsStaff bool      `json:"ftIsStaff" example:"true"`
	PhotoURL  string    `json:"url" example:"https://intra.42.fr/some-login/some-id"`
	LastSeen  time.Time `json:"lastUpdate" example:"2025-02-18T15:00:00Z"`
	IsStaff   bool      `json:"isStaff" example:"true"`
}

type RolePatch struct {
	ID    string  `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name  *string `json:"name" example:"captain-hook"`
	Color *string `json:"color" example:"0xFF00FF"`
}

type ModulePatch struct {
	ID            string     `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	Name          *string    `json:"name" example:"captain-hook"`
	Version       *string    `json:"version" example:"1.2"`
	Status        *string    `json:"status" example:"enabled"`
	URL           *string    `json:"url" example:"https://github.com/some-user/some-repo"`
	LatestVersion *string    `json:"lastestVersion" example:"1.7"`
	LateCommits   *int       `json:"lateCommits" example:"2"`
	LastUpdate    *time.Time `json:"lastUpdate" example:"2025-02-18T15:00:00Z"`
}

type UserPatch struct {
	ID        string     `json:"id" example:"01HZ0MMK4S6VQW4WPHB6NZ7R7X"`
	FtLogin   *string    `json:"login" example:"heinz"`
	FtID      *string    `json:"ftId" example:"1492"`
	FtIsStaff *bool      `json:"ftIsStaff" example:"true"`
	PhotoURL  *string    `json:"url" example:"https://intra.42.fr/some-login/some-id"`
	LastSeen  *time.Time `json:"lastUpdate" example:"2025-02-18T15:00:00Z"`
	IsStaff   *bool      `json:"isStaff" example:"true"`
}

var mainDB *sql.DB
var testDB *sql.DB

func init() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	mainDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Couldn't connect to database: ", err)
	}
}
