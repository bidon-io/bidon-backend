package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/bidon-io/bidon-backend/config"
	"github.com/bidon-io/bidon-backend/internal/db"
	"github.com/bidon-io/bidon-backend/internal/db/dbtest"
)

func TestAdUnitLookup_GetByUID(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	lineItem1 := dbtest.CreateLineItem(t, tx, func(item *db.LineItem) {
		item.PublicUID.Int64 = 12345
		item.PublicUID.Valid = true
		item.Extra = map[string]any{"test_key": "test_value"}
	})

	lineItem2 := dbtest.CreateLineItem(t, tx, func(item *db.LineItem) {
		item.PublicUID.Int64 = 67890
		item.PublicUID.Valid = true
		item.Extra = map[string]any{"another_key": "another_value"}
	})

	// Pure database method - no cache needed
	lookup := &AdUnitLookup{DB: tx}

	tests := []struct {
		name          string
		uid           string
		expectedID    int64
		expectedExtra map[string]any
		expectNil     bool
	}{
		{
			name:          "valid UID returns correct LineItem",
			uid:           "12345",
			expectedID:    lineItem1.ID,
			expectedExtra: map[string]any{"test_key": "test_value"},
			expectNil:     false,
		},
		{
			name:          "different UID returns different LineItem",
			uid:           "67890",
			expectedID:    lineItem2.ID,
			expectedExtra: map[string]any{"another_key": "another_value"},
			expectNil:     false,
		},
		{
			name:      "invalid UID returns nil",
			uid:       "invalid",
			expectNil: true,
		},
		{
			name:      "empty UID returns nil",
			uid:       "",
			expectNil: true,
		},
		{
			name:      "non-existent UID returns nil",
			uid:       "99999",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lookup.GetByUID(context.Background(), tt.uid)

			if err != nil {
				t.Fatalf("GetByUID() error = %v", err)
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("GetByUID() expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Fatalf("GetByUID() expected non-nil result")
				}

				if diff := cmp.Diff(tt.expectedID, result.ID); diff != "" {
					t.Errorf("GetByUID() ID mismatch (-want +got):\n%s", diff)
				}

				if diff := cmp.Diff(tt.expectedExtra, result.Extra); diff != "" {
					t.Errorf("GetByUID() Extra mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAdUnitLookup_GetByUIDCached(t *testing.T) {
	tx := testDB.Begin()
	defer tx.Rollback()

	lineItem1 := dbtest.CreateLineItem(t, tx, func(item *db.LineItem) {
		item.PublicUID.Int64 = 12345
		item.PublicUID.Valid = true
		item.Extra = map[string]any{"test_key": "test_value"}
	})

	lineItem2 := dbtest.CreateLineItem(t, tx, func(item *db.LineItem) {
		item.PublicUID.Int64 = 67890
		item.PublicUID.Valid = true
		item.Extra = map[string]any{"another_key": "another_value"}
	})

	tests := []struct {
		name          string
		uid           string
		expectedID    int64
		expectedExtra map[string]any
		expectNil     bool
	}{
		{
			name:          "valid UID returns correct LineItem",
			uid:           "12345",
			expectedID:    lineItem1.ID,
			expectedExtra: map[string]any{"test_key": "test_value"},
			expectNil:     false,
		},
		{
			name:          "different UID returns different LineItem",
			uid:           "67890",
			expectedID:    lineItem2.ID,
			expectedExtra: map[string]any{"another_key": "another_value"},
			expectNil:     false,
		},
		{
			name:      "invalid UID returns nil",
			uid:       "invalid",
			expectNil: true,
		},
		{
			name:      "empty UID returns nil",
			uid:       "",
			expectNil: true,
		},
		{
			name:      "non-existent UID returns nil",
			uid:       "99999",
			expectNil: true,
		},
	}

	adUnitLookupCache := config.NewMemoryCacheOf[*db.LineItem](time.Minute)
	lookup := &AdUnitLookup{
		DB:    tx,
		Cache: adUnitLookupCache,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lookup.GetByUIDCached(context.Background(), tt.uid)

			if err != nil {
				t.Fatalf("GetByUIDCached() error = %v", err)
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("GetByUIDCached() expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Fatalf("GetByUIDCached() expected non-nil result")
				}

				if diff := cmp.Diff(tt.expectedID, result.ID); diff != "" {
					t.Errorf("GetByUIDCached() ID mismatch (-want +got):\n%s", diff)
				}

				if diff := cmp.Diff(tt.expectedExtra, result.Extra); diff != "" {
					t.Errorf("GetByUIDCached() Extra mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
