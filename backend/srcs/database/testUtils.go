package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var mainDB *sqlx.DB
var testDB *sqlx.DB

func init() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	mainDB, err = sqlx.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Couldn't connect to database: ", err)
	}
}

func formatModules(ms []Module) string {
	var b strings.Builder
	if ms == nil {
		return "nil"
	}
	b.WriteString("[]Module{\n")
	for _, m := range ms {
		// extract the components of LastUpdate
		y, mon, d := m.LastUpdate.Date()
		h, min, sec := m.LastUpdate.Clock()
		b.WriteString(fmt.Sprintf(
			"\t{ID: %q, Name: %q, Version: %q, Status: %q, GitURL: %q, IconURL: %q, LatestVersion: %q, LateCommits: %d, LastUpdate: time.Date(%d, %d, %d, %d, %d, %d, 0, time.UTC)},\n",
			m.ID, m.Name, m.Version, m.Status, m.GitURL, m.IconURL, m.LatestVersion, m.LateCommits,
			y, int(mon), d, h, min, sec,
		))
	}
	b.WriteString("},")
	return b.String()
}

func formatUsers(us []User) string {
	var b strings.Builder
	if us == nil {
		return "nil"
	}
	b.WriteString("[]User{\n")
	for _, u := range us {
		y, mon, d := u.LastSeen.Date()
		h, min, sec := u.LastSeen.Clock()
		b.WriteString(fmt.Sprintf(
			"\t{ID: %q, FtLogin: %q, FtID: %q, FtIsStaff: %t, PhotoURL: %q, LastSeen: time.Date(%d, %d, %d, %d, %d, %d, 0, time.UTC), IsStaff: %t},\n",
			u.ID, u.FtLogin, u.FtID, u.FtIsStaff, u.PhotoURL,
			y, int(mon), d, h, min, sec,
			u.IsStaff,
		))
	}
	b.WriteString("},")
	return b.String()
}

func formatRoles(rs []Role) string {
	var b strings.Builder
	if rs == nil {
		return "nil"
	}
	b.WriteString("[]Role{\n")
	for _, r := range rs {
		b.WriteString(fmt.Sprintf(
			"\t{ID: %q, Name: %q, Color: %q},\n",
			r.ID, r.Name, r.Color,
		))
	}
	b.WriteString("},")
	return b.String()
}

func DropDatabase(t *testing.T, dbName string) {
	_, err := mainDB.Exec(fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", dbName))
	if err != nil {
		t.Fatalf("failed to drop database %s: %v", dbName, err)
	}
}

func CreateAndPopulateDatabase(t *testing.T, dbName string, sqlFile string) *sqlx.DB {
	_, err := mainDB.Exec(fmt.Sprintf(
		"DROP DATABASE IF EXISTS %s WITH (FORCE)",
		dbName,
	))
	if err != nil {
		t.Fatalf("failed to drop test database %s: %v", dbName, err)
	}

	_, err = mainDB.Exec(fmt.Sprintf(
		"CREATE DATABASE %s TEMPLATE schema_template",
		dbName,
	))
	if err != nil {
		t.Fatalf("failed to create test database %s: %v", dbName, err)
	}

	host := os.Getenv("HOST_NAME")
	if host == "" {
		host = "localhost"
	}

	testDB, err = sqlx.Open("postgres", fmt.Sprintf("postgresql://admin:pw_admin@%s/%s?sslmode=disable", host, dbName))
	if err != nil {
		t.Fatalf("failed to connect to test database %s: %v", dbName, err)
	}

	_, err = testDB.Exec(sqlFile)
	if err != nil {
		t.Fatalf("failed to execute SQL in %s: %v", sqlFile, err)
	}

	tmp := mainDB
	mainDB = testDB
	testDB = tmp
	t.Cleanup(func() {
		tmp := mainDB
		mainDB = testDB
		testDB = tmp
		DropDatabase(t, dbName)
	})

	return mainDB
}
