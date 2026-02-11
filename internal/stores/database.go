package stores

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connect(databaseUrl string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseUrl)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	return db, nil
}
