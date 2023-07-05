package testutils

import (
	"context"
	"fmt"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/admin/store"
	"github.com/bidon-io/bidon-backend/internal/db"
)

func CreateUser(t *testing.T, tx *db.DB, index int) *admin.User {
	t.Helper()

	userRepo := store.NewUserRepo(tx)
	userAttrs := &admin.UserAttrs{
		Email: fmt.Sprintf("test%d@email.com", index),
	}

	user, err := userRepo.Create(context.Background(), userAttrs)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	return user
}

func CreateUsersList(t *testing.T, tx *db.DB, usersCount int) []*admin.User {
	t.Helper()

	users := make([]*admin.User, usersCount)
	for i := range users {
		users[i] = CreateUser(t, tx, i)
	}

	return users
}
