package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type Connection struct {
	DB *sql.DB
}

func NewDatabaseConnection(host, port, user, password, dbname string) (*Connection, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Connection{DB: db}, nil
}
