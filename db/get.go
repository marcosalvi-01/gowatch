package db

import (
	"database/sql"
	_ "embed"
	"gowatch/logging"
	"os"
	"path"

	_ "github.com/mattn/go-sqlite3"
)

//go:generate go tool sqlc generate -f ../sqlc.yaml

//go:embed schema.sql
var schema string

var log = logging.Get("database")

// open a new connection to the db. if it does not exist in the file system it is created.
func Get(DBPath, DBName string) (*Queries, error) {
	err := os.MkdirAll(DBPath, 0755)
	if err != nil {
		return nil, err
	}

	dbFile := path.Join(DBPath, DBName)

	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		file, err := os.Create(dbFile)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	queries := New(db)

	return queries, nil
}
