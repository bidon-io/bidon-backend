// Package dbtest provides helper functions for tests that require database access
package dbtest

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/joho/godotenv"
)

func Prepare() *db.DB {
	var (
		testDB *db.DB
		err    error
	)

	err = godotenv.Load("../../../.env.test")
	if err != nil {
		log.Printf("Did not load from .env.test file: %v", err)
	}

	testDB, err = db.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	err = testDB.AutoMigrate()
	if err != nil {
		log.Fatalf("Error migrating the database: %v", err)
	}

	return testDB
}

func CreateUser(ctx context.Context, tx *db.DB, index int) *db.User {
	user := &db.User{
		Email: fmt.Sprintf("test%d@email.com", index),
	}

	if err := tx.WithContext(ctx).Create(user).Error; err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
	return user
}

func CreateUsersList(ctx context.Context, tx *db.DB, usersCount int) []*db.User {
	users := make([]*db.User, usersCount)
	for i := range users {
		users[i] = CreateUser(ctx, tx, i)
	}

	return users
}
