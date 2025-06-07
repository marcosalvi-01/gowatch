package db

import (
	"database/sql"
	_ "embed"
	"gowatch/logging"
	"os"
	"path"

	_ "modernc.org/sqlite"
)

//go:generate go tool sqlc generate -f ../sqlc.yaml

//go:embed schema.sql
var schema string

var log = logging.Get("database")

// open a new connection to the db. if it does not exist in the file system it is created.
func Get(DBPath, DBName string) (*Queries, error) {
	log.Debug("Ensuring DB path exists", "path", DBPath)
	err := os.MkdirAll(DBPath, 0755)
	if err != nil {
		log.Error("Failed to create DB path", "path", DBPath, "error", err)
		return nil, err
	}

	dbFile := path.Join(DBPath, DBName)
	log.Debug("Resolved DB file path", "dbFile", dbFile)

	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		log.Info("DB file does not exist, creating", "dbFile", dbFile)
		file, err := os.Create(dbFile)
		if err != nil {
			log.Error("Failed to create DB file", "dbFile", dbFile, "error", err)
			return nil, err
		}
		file.Close()
	} else {
		log.Debug("DB file already exists", "dbFile", dbFile)
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Error("Failed to open DB connection", "dbFile", dbFile, "error", err)
		return nil, err
	}
	log.Debug("DB connection opened", "dbFile", dbFile)

	_, err = db.Exec(schema)
	if err != nil {
		log.Error("Failed to execute schema", "error", err)
		return nil, err
	}
	log.Debug("Schema executed successfully")

	queries := New(db)
	log.Info("DB initialized successfully", "dbFile", dbFile)

	return queries, nil
}
