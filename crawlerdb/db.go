package crawlerdb

import (
	"github.com/jmoiron/sqlx"
)

// Postgres represents a Postgres database client.
type Postgres struct {
	db *sqlx.DB
}

// New creates a new Postgres client.
func New(dsn string) (*Postgres, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &Postgres{db: db}, err
}
