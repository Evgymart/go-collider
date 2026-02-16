package utils

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

func CreateTestUser(db *sqlx.DB) int64 {
	query := `insert into users (name) values ($1) returning user_id`
	var userID int64
	if err := db.Get(&userID, query, "test user"); err != nil {
		panic(fmt.Sprintf("failed to create test user: %v", err))
	}
	return userID
}
