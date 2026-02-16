package stores

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connect(databaseUrl string) (*sqlx.DB, error) {
	return ConnectWithPool(databaseUrl, 20, 10)
}

func ConnectWithPool(databaseUrl string, maxOpenConns, maxIdleConns int) (*sqlx.DB, error) {
	if maxOpenConns <= 0 {
		return nil, fmt.Errorf("maxOpenConns must be positive, got %d", maxOpenConns)
	}
	if maxIdleConns <= 0 {
		return nil, fmt.Errorf("maxIdleConns must be positive, got %d", maxIdleConns)
	}
	if maxIdleConns > maxOpenConns {
		return nil, fmt.Errorf("maxIdleConns (%d) cannot exceed maxOpenConns (%d)", maxIdleConns, maxOpenConns)
	}

	db, err := sqlx.Connect("postgres", databaseUrl)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	return db, nil
}
