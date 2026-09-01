package dspsim

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bidon-io/bidon-backend/internal/ad"
)

// Notification kinds recorded against a bid.
const (
	NotificationWin     = "win"     // nurl
	NotificationLoss    = "loss"    // lurl
	NotificationBilling = "billing" // burl
	NotificationImp     = "creative_impression"
	NotificationClick   = "creative_click"
	NotificationTrack   = "creative_track"
)

// Notification is one inbound hit on a URL the simulator advertised.
type Notification struct {
	Kind       string            `json:"kind"`
	Event      string            `json:"event,omitempty"`
	ReceivedAt time.Time         `json:"received_at"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	RawQuery   string            `json:"raw_query,omitempty"`
	Params     map[string]string `json:"params,omitempty"`
	// UnresolvedMacros lists params whose value still contains a literal
	// ${MACRO}, meaning the sender did not substitute it.
	UnresolvedMacros []string `json:"unresolved_macros,omitempty"`
	RemoteAddr       string   `json:"remote_addr,omitempty"`
	UserAgent        string   `json:"user_agent,omitempty"`
	// BidID is only set on orphan notifications, where no record matched.
	BidID string `json:"bid_id,omitempty"`
}

// BidRecord is everything the simulator remembers about one bid it made.
type BidRecord struct {
	BidID            string    `json:"bid_id"`
	RequestID        string    `json:"request_id"`
	ImpID            string    `json:"imp_id"`
	Bundle           string    `json:"bundle"`
	DSP              string    `json:"dsp"`
	AdType           ad.Type   `json:"ad_type"`
	Format           ad.Format `json:"format,omitempty"`
	Width            int64     `json:"w,omitempty"`
	Height           int64     `json:"h,omitempty"`
	Floor            float64   `json:"floor"`
	Price            float64   `json:"price"`
	Currency         string    `json:"currency"`
	CreativeID       string    `json:"creative_id"`
	CreativeType     string    `json:"creative_type"`
	CreativeBucket   string    `json:"creative_bucket"`
	AuctionConfigID  int64     `json:"auction_config_id"`
	DemandConfigured bool      `json:"demand_configured"`
	CreatedAt        time.Time `json:"created_at"`
	NURL             string    `json:"nurl"`
	BURL             string    `json:"burl"`
	LURL             string    `json:"lurl"`

	Notifications []Notification `json:"notifications"`
}

// Store keeps bid records in memory, keyed by bid id. Records expire after a
// TTL and the oldest are evicted once the cap is reached. Nothing is persisted:
// the simulator is a test instrument, not a system of record.
type Store struct {
	ttl     time.Duration
	max     int
	orphans int

	mu           sync.RWMutex
	records      map[string]*BidRecord
	order        []string
	orphanEvents []Notification
	now          func() time.Time
}

// NewStore returns a store retaining records for ttl, capped at max entries.
func NewStore(ttl time.Duration, max int) *Store {
	if max <= 0 {
		max = 10_000
	}
	return &Store{
		ttl:     ttl,
		max:     max,
		orphans: 200,
		records: map[string]*BidRecord{},
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Put stores a bid record, evicting expired and surplus entries.
func (s *Store) Put(rec *BidRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[rec.BidID]; !exists {
		s.order = append(s.order, rec.BidID)
	}
	s.records[rec.BidID] = rec

	s.evictLocked()
}

// Record appends a notification to a bid record. It reports whether a record
// with that id existed; an unknown id is kept in the orphan list instead.
func (s *Store) Record(bidID string, n Notification) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.records[bidID]
	if !ok {
		n.BidID = bidID
		s.orphanEvents = append(s.orphanEvents, n)
		if len(s.orphanEvents) > s.orphans {
			s.orphanEvents = s.orphanEvents[len(s.orphanEvents)-s.orphans:]
		}
		return false
	}

	rec.Notifications = append(rec.Notifications, n)
	return true
}

// Get returns a copy of the record for bidID.
func (s *Store) Get(bidID string) (*BidRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.records[bidID]
	if !ok {
		return nil, false
	}
	return copyRecord(rec), true
}

// List returns the retained records, newest first, optionally filtered by DSP.
func (s *Store) List(dsp string) []*BidRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dsp = strings.ToLower(strings.TrimSpace(dsp))
	out := make([]*BidRecord, 0, len(s.records))
	for _, id := range s.order {
		rec, ok := s.records[id]
		if !ok {
			continue
		}
		if dsp != "" && strings.ToLower(rec.DSP) != dsp {
			continue
		}
		out = append(out, copyRecord(rec))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Orphans returns notifications received for unknown bid ids.
func (s *Store) Orphans() []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Notification, len(s.orphanEvents))
	copy(out, s.orphanEvents)
	return out
}

// Len returns the number of retained records.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// Clear drops every record and orphan.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = map[string]*BidRecord{}
	s.order = nil
	s.orphanEvents = nil
}

// evictLocked removes expired records first, then the oldest ones until the
// store is within its cap. Callers must hold the write lock.
func (s *Store) evictLocked() {
	if s.ttl > 0 {
		cutoff := s.now().Add(-s.ttl)
		kept := s.order[:0]
		for _, id := range s.order {
			rec, ok := s.records[id]
			if !ok {
				continue
			}
			if rec.CreatedAt.Before(cutoff) {
				delete(s.records, id)
				continue
			}
			kept = append(kept, id)
		}
		s.order = kept
	}

	for len(s.order) > s.max {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.records, oldest)
	}
}

func copyRecord(rec *BidRecord) *BidRecord {
	out := *rec
	out.Notifications = make([]Notification, len(rec.Notifications))
	copy(out.Notifications, rec.Notifications)
	return &out
}
