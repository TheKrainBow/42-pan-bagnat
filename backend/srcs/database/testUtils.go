package database

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

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
			"\t{ID: %q, Name: %q, Version: %q, Status: %q, URL: %q, LatestVersion: %q, LateCommits: %d, LastUpdate: time.Date(%d, %d, %d, %d, %d, %d, 0, time.UTC)},\n",
			m.ID, m.Name, m.Version, m.Status, m.URL, m.LatestVersion, m.LateCommits,
			y, int(mon), d, h, min, sec,
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

func CreateAndPopulateDatabase(t *testing.T, dbName string, sqlFile string) *sql.DB {
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

	testDB, err = sql.Open("postgres", fmt.Sprintf("postgresql://admin:pw_admin@localhost/%s?sslmode=disable", dbName))
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
