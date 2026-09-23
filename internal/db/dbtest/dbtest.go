// Package dbtest provides helper functions for tests that require database access
package dbtest

import (
	"hash/fnv"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/bwmarrin/snowflake"

	"github.com/bidon-io/bidon-backend/internal/db"
)

// defaultDatabaseURL points at the postgres-test service published by docker-compose.yml.
const defaultDatabaseURL = "postgres://bidon:pass@localhost:5435/bidon_test"

func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return defaultDatabaseURL
}

// Prepare sets up the database for testing and initializes test factories.
// It should be called once per test package, usually in TestMain.
func Prepare() *db.DB {
	return prepare(databaseURL(), callerHash())
}

// PrepareURL is like Prepare, but opens url instead of DATABASE_URL. Use it for
// a test package that needs its own database (e.g. the e2e suite, which
// commits fixtures a rolled-back transaction couldn't share with a
// separate process).
func PrepareURL(url string) *db.DB {
	return prepare(url, callerHash())
}

// callerHash hashes the file path of Prepare/PrepareURL's caller, to use as a
// snowflake node ID and base for factory counters. Because running tests from
// multiple packages in parallel can cause transaction deadlocks.
func callerHash() uint32 {
	var hashNum uint32
	if _, file, _, ok := runtime.Caller(2); ok {
		hash := fnv.New32()
		_, _ = hash.Write([]byte(file))
		hashNum = hash.Sum32()
	}
	return hashNum
}

func prepare(url string, hashNum uint32) *db.DB {
	counter.base = hashNum >> 16 // cut in half for readability

	nodeID := int64(hashNum >> 22) // max value is 1023 or 10 bits
	node, err := snowflake.NewNode(nodeID)
	if err != nil {
		log.Fatalf("Error creating snowflake node: %v", err)
	}

	testDB, err := db.Open(
		url,
		db.WithIgnoreRecordNotFoundError(),
		db.WithSnowflakeNode(node),
	)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	return testDB
}

var counter = newCounters()

type counters struct {
	mu       sync.Mutex
	counters map[string]uint32
	base     uint32
}

func newCounters() *counters {
	return &counters{
		counters: make(map[string]uint32),
	}
}

func (c *counters) get(name string) uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.counters[name]; ok {
		c.counters[name]++
	} else {
		c.counters[name] = c.base
	}

	return c.counters[name]
}
