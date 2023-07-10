// Package dbtest provides helper functions for tests that require database access
package dbtest

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/segment"
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

func CreateApp(t *testing.T, tx *db.DB, index int, user *db.User) *db.App {
	t.Helper()

	if user == nil {
		user = CreateUser(t, tx, index)
	}

	app := &db.App{
		UserID:     user.ID,
		PlatformID: db.AndroidPlatformID,
		HumanName:  "Test App",
		PackageName: sql.NullString{
			String: fmt.Sprintf("app.package%d", index),
			Valid:  true,
		},
		AppKey: sql.NullString{
			String: fmt.Sprintf("appkey%d", index),
			Valid:  true,
		},
		Settings: make(map[string]any),
	}

	if err := tx.Create(app).Error; err != nil {
		t.Fatalf("Failed to create app: %v", app)
	}
	return app
}

func CreateAppsList(t *testing.T, tx *db.DB, count int) []*db.App {
	t.Helper()

	apps := make([]*db.App, count)
	for i := range apps {
		apps[i] = CreateApp(t, tx, i, nil)
	}

	return apps
}

func CreateDemandSource(t *testing.T, tx *db.DB, index int) *db.DemandSource {
	t.Helper()

	demandSource := &db.DemandSource{
		APIKey:    fmt.Sprintf("apikey%d", index),
		HumanName: "demandsource",
	}

	if err := tx.Create(demandSource).Error; err != nil {
		t.Fatalf("Failed to create demand source: %v", err)
	}
	return demandSource
}

func CreateDemandSourcesList(t *testing.T, tx *db.DB, count int) []*db.DemandSource {
	t.Helper()

	demandSources := make([]*db.DemandSource, count)
	for i := range demandSources {
		demandSources[i] = CreateDemandSource(t, tx, i)
	}

	return demandSources
}

func CreateDemandSourceAccount(t *testing.T, tx *db.DB, index int, user *db.User, demandSource *db.DemandSource) *db.DemandSourceAccount {
	t.Helper()

	if user == nil {
		user = CreateUser(t, tx, index)
	}

	if demandSource == nil {
		demandSource = CreateDemandSource(t, tx, index)
	}

	account := &db.DemandSourceAccount{
		DemandSourceID: demandSource.ID,
		UserID:         user.ID,
		Type:           "DemandSourceAccount::Admob",
		Extra:          map[string]any{},
		IsBidding:      new(bool),
		IsDefault: sql.NullBool{
			Valid: true,
			Bool:  true,
		},
	}

	if err := tx.Create(account).Error; err != nil {
		t.Fatalf("Failed to create demand source account: %v", err)
	}
	return account
}

func CreateDemandSourceAccountsList(t *testing.T, tx *db.DB, count int) []*db.DemandSourceAccount {
	t.Helper()

	accounts := make([]*db.DemandSourceAccount, count)
	for i := range accounts {
		accounts[i] = CreateDemandSourceAccount(t, tx, i, nil, nil)
	}

	return accounts
}

func CreateSegment(t *testing.T, tx *db.DB, index int, app *db.App) *db.Segment {
	t.Helper()

	if app == nil {
		app = CreateApp(t, tx, index, nil)
	}

	segment := &db.Segment{
		Name:        "Test Segment",
		Description: "description",
		Filters:     []segment.Filter{},
		Enabled:     new(bool),
		AppID:       app.ID,
		Priority:    0,
	}

	if err := tx.Create(segment).Error; err != nil {
		t.Fatalf("Failed to create segment: %v", err)
	}
	return segment
}

func CreateSegmentsList(t *testing.T, tx *db.DB, count int) []*db.Segment {
	t.Helper()

	segments := make([]*db.Segment, count)
	for i := range segments {
		segments[i] = CreateSegment(t, tx, i, nil)
	}

	return segments
}

func CreateUser(t *testing.T, tx *db.DB, index int) *db.User {
	t.Helper()

	user := &db.User{
		Email: fmt.Sprintf("test%d@email.com", index),
	}

	if err := tx.Create(user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	return user
}

func CreateUsersList(t *testing.T, tx *db.DB, usersCount int) []*db.User {
	t.Helper()

	users := make([]*db.User, usersCount)
	for i := range users {
		users[i] = CreateUser(t, tx, i)
	}

	return users
}
