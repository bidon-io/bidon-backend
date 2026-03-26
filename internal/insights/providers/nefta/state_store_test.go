package nefta

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
)

func TestRedisStateStoreFindNotFound(t *testing.T) {
	rdb, mock := redismock.NewClusterMock()
	mock.ExpectGet("nefta:1:idfa").RedisNil()

	store := NewRedisStateStore(rdb)
	got, err := store.Find(context.Background(), "nefta:1:idfa")
	if err != nil {
		t.Fatalf("find state: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil state, got %+v", got)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestRedisStateStoreFindAndSave(t *testing.T) {
	rdb, mock := redismock.NewClusterMock()
	mock.ExpectGet("nefta:2:idfa").SetVal(`{"nuid":"abc","session_id":3,"ad_opportunity_id":0,"last_activity_ts":10,"session_start_ts":5}`)
	mock.ExpectSet("nefta:2:idfa", `{"nuid":"def","session_id":4,"ad_opportunity_id":1,"last_activity_ts":11,"session_start_ts":11}`, 30*24*time.Hour).SetVal("OK")

	store := NewRedisStateStore(rdb)

	got, err := store.Find(context.Background(), "nefta:2:idfa")
	if err != nil {
		t.Fatalf("find state: %v", err)
	}
	if got == nil || got.NUID != "abc" || got.SessionID != 3 {
		t.Fatalf("unexpected state: %+v", got)
	}

	err = store.Save(context.Background(), "nefta:2:idfa", &State{
		NUID:            "def",
		SessionID:       4,
		AdOpportunityID: 1,
		LastActivityTS:  11,
		SessionStartTS:  11,
	})
	if err != nil {
		t.Fatalf("save state: %v", err)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
