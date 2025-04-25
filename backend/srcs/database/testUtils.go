package database

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/lib/pq"
)

func DropDatabase(t *testing.T, dbName string) {
	_, err := mainDB.Exec(fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", dbName))
	if err != nil {
		t.Fatalf("failed to drop database %s: %v", dbName, err)
	}
}

func CreateAndPopulateDatabase(t *testing.T, dbName string, sqlFile string) *sql.DB {
	_, err := mainDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
	if err != nil {
		t.Fatalf("failed to drop database %s: %v", dbName, err)
	}

	_, err = mainDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("failed to create database %s: %v", dbName, err)
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
