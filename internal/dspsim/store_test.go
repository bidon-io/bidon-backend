package dspsim

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func testRecord(id string, createdAt time.Time) *BidRecord {
	return &BidRecord{BidID: id, DSP: "adikteev", CreatedAt: createdAt}
}

func TestStoreRecordsNotifications(t *testing.T) {
	store := NewStore(time.Hour, 10)
	store.Put(testRecord("bid-1", time.Now().UTC()))

	if known := store.Record("bid-1", Notification{Kind: NotificationWin}); !known {
		t.Fatal("Record() = false for a known bid id")
	}

	record, ok := store.Get("bid-1")
	if !ok {
		t.Fatal("Get() = false after Put()")
	}
	if len(record.Notifications) != 1 || record.Notifications[0].Kind != NotificationWin {
		t.Fatalf("notifications = %+v, want one win", record.Notifications)
	}

	// The returned record is a copy: mutating it must not affect the store.
	record.Notifications = append(record.Notifications, Notification{Kind: NotificationLoss})
	if again, _ := store.Get("bid-1"); len(again.Notifications) != 1 {
		t.Errorf("Get() returned a shared slice, store now has %d notifications", len(again.Notifications))
	}
}

func TestStoreOrphanNotifications(t *testing.T) {
	store := NewStore(time.Hour, 10)

	if known := store.Record("never-seen", Notification{Kind: NotificationBilling}); known {
		t.Fatal("Record() = true for an unknown bid id")
	}

	orphans := store.Orphans()
	if len(orphans) != 1 {
		t.Fatalf("Orphans() = %d, want 1", len(orphans))
	}
	if orphans[0].BidID != "never-seen" {
		t.Errorf("orphan BidID = %q, want never-seen", orphans[0].BidID)
	}
}

func TestStoreEvictsExpiredRecords(t *testing.T) {
	store := NewStore(time.Minute, 10)

	store.Put(testRecord("old", time.Now().UTC().Add(-2*time.Minute)))
	store.Put(testRecord("fresh", time.Now().UTC()))

	if _, ok := store.Get("old"); ok {
		t.Error("expired record was not evicted")
	}
	if _, ok := store.Get("fresh"); !ok {
		t.Error("fresh record was evicted")
	}
}

func TestStoreEvictsOldestOverCap(t *testing.T) {
	store := NewStore(time.Hour, 3)

	now := time.Now().UTC()
	for i := range 5 {
		store.Put(testRecord(fmt.Sprintf("bid-%d", i), now.Add(time.Duration(i)*time.Second)))
	}

	if got := store.Len(); got != 3 {
		t.Fatalf("Len() = %d, want 3", got)
	}
	for _, evicted := range []string{"bid-0", "bid-1"} {
		if _, ok := store.Get(evicted); ok {
			t.Errorf("%s should have been evicted", evicted)
		}
	}
	for _, kept := range []string{"bid-2", "bid-3", "bid-4"} {
		if _, ok := store.Get(kept); !ok {
			t.Errorf("%s should have been kept", kept)
		}
	}
}

func TestStoreListFiltersByDSPNewestFirst(t *testing.T) {
	store := NewStore(time.Hour, 10)

	now := time.Now().UTC()
	older := testRecord("older", now.Add(-time.Second))
	newer := testRecord("newer", now)
	other := testRecord("other", now)
	other.DSP = "meta"

	store.Put(older)
	store.Put(newer)
	store.Put(other)

	all := store.List("")
	if len(all) != 3 {
		t.Fatalf("List() = %d records, want 3", len(all))
	}
	if all[0].BidID == "older" {
		t.Error("List() should return the newest record first")
	}

	adikteev := store.List("ADIKTEEV")
	if len(adikteev) != 2 {
		t.Fatalf("List(adikteev) = %d records, want 2", len(adikteev))
	}
}

func TestStoreClear(t *testing.T) {
	store := NewStore(time.Hour, 10)
	store.Put(testRecord("bid-1", time.Now().UTC()))
	store.Record("unknown", Notification{Kind: NotificationWin})

	store.Clear()

	if store.Len() != 0 {
		t.Errorf("Len() = %d after Clear()", store.Len())
	}
	if len(store.Orphans()) != 0 {
		t.Errorf("Orphans() = %d after Clear()", len(store.Orphans()))
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	store := NewStore(time.Hour, 1_000)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			id := fmt.Sprintf("bid-%d", i)
			store.Put(testRecord(id, time.Now().UTC()))
			store.Record(id, Notification{Kind: NotificationWin})
			store.Get(id)
			store.List("")
			store.Orphans()
		}(i)
	}
	wg.Wait()

	if got := store.Len(); got != 50 {
		t.Errorf("Len() = %d, want 50", got)
	}
}
