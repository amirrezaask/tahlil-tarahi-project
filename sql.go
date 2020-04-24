package main

import (
	"database/sql"
)

var db *sql.DB = nil

func connectDb() (*sql.DB, error) {
	if err := db.Ping; err == nil {
		return db, nil
	}
	db, err := sql.Open("sqlite3", "db.sqlite3")
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func dbQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func dbExec(stmt string, args ...interface{}) (sql.Result, error) {
	return db.Exec(stmt, args...)
}
