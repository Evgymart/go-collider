package utils

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func CreateTestUser(db *sqlx.DB) uuid.UUID {
	userID := uuid.New()
	query := `insert into users (user_id, name) values ($1, $2)`
	if _, err := db.Exec(query, userID, "test user"); err != nil {
		panic(fmt.Sprintf("failed to create test user: %v", err))
	}
	return userID
}
