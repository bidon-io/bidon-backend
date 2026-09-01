// Package dspsim implements a standalone OpenRTB DSP simulator.
//
// The simulator speaks OpenRTB 2.x over HTTP, reads auction configuration from
// the shared Bidon Postgres schema, answers bid requests with a random creative
// picked from a JSON library indexed by DSP and creative type, and receives the
// nurl / burl / lurl notifications it advertises. Every interaction is logged
// and kept in memory, keyed by bid id, so a full auction lifecycle can be
// inspected after the fact.
//
// Nothing in bidon is modified to support the simulator. Adapter RTB endpoints
// are hardcoded in Go (except Bidmachine, which reads "endpoint" from
// demand_source_accounts.extra), so pointing a live bidon auction at the
// simulator requires an out-of-band redirect: a hosts entry, an HTTP proxy, or
// a Bidmachine endpoint override in the database. See
// docs/adr/0001-dsp-simulator.md.
package dspsim

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every knob of the simulator. All fields are populated from the
// environment by LoadConfig.
type Config struct {
	// DatabaseURL is the read-only connection to the Bidon schema.
	DatabaseURL string
	// Port is the HTTP listen port.
	Port string
	// PublicURL is the base URL the simulator advertises in notification and
	// creative URLs. It must be reachable by whoever receives the bid.
	PublicURL string
	// CreativesFile points at a JSON creative library. Empty means the
	// embedded default library.
	CreativesFile string

	// Seat is the seat id reported in seatbid[0].seat.
	Seat string
	// Currency is the bid response currency.
	Currency string

	// PriceMultMin and PriceMultMax bound the random multiplier applied to the
	// impression bid floor.
	PriceMultMin float64
	PriceMultMax float64
	// FallbackFloor is used as the floor when the impression carries none.
	FallbackFloor float64
	// MaxPrice caps the generated price. An impression floor above it is a
	// no-bid with reason ReasonFloorTooHigh.
	MaxPrice float64

	// NoBidRate is the probability in [0,1] of answering 204 to an otherwise
	// matched request.
	NoBidRate float64
	// Latency is an artificial delay applied before answering a bid request.
	Latency time.Duration

	// CatalogTTL is the interval between catalog refreshes from Postgres.
	CatalogTTL time.Duration
	// BidTTL is how long a bid record is retained in memory.
	BidTTL time.Duration
	// MaxBids caps the number of retained bid records, oldest evicted first.
	MaxBids int

	// StrictDemand turns the "is this DSP actually configured as bidding demand
	// for this app?" check from a warning into a no-bid.
	StrictDemand bool
	// Seed makes creative selection and pricing deterministic. Zero means a
	// time-based seed.
	Seed int64
}

// LoadConfig reads the simulator configuration from the environment.
func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		Port:          envString("DSPSIM_PORT", "1325"),
		PublicURL:     envString("DSPSIM_PUBLIC_URL", ""),
		CreativesFile: os.Getenv("DSPSIM_CREATIVES_FILE"),
		Seat:          envString("DSPSIM_SEAT", "dspsim"),
		Currency:      envString("DSPSIM_CUR", "USD"),
	}

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("missing DATABASE_URL environment variable")
	}
	if cfg.PublicURL == "" {
		cfg.PublicURL = fmt.Sprintf("http://localhost:%s", cfg.Port)
	}
	cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")

	var err error
	if cfg.PriceMultMin, err = envFloat("DSPSIM_PRICE_MULT_MIN", 1.5); err != nil {
		return cfg, err
	}
	if cfg.PriceMultMax, err = envFloat("DSPSIM_PRICE_MULT_MAX", 3.0); err != nil {
		return cfg, err
	}
	if cfg.FallbackFloor, err = envFloat("DSPSIM_FALLBACK_FLOOR", 0.5); err != nil {
		return cfg, err
	}
	if cfg.MaxPrice, err = envFloat("DSPSIM_MAX_PRICE", 25.0); err != nil {
		return cfg, err
	}
	if cfg.NoBidRate, err = envFloat("DSPSIM_NO_BID_RATE", 0); err != nil {
		return cfg, err
	}
	if cfg.Latency, err = envDuration("DSPSIM_LATENCY_MS", 0, time.Millisecond); err != nil {
		return cfg, err
	}
	if cfg.CatalogTTL, err = envDuration("DSPSIM_CATALOG_TTL", 60, time.Second); err != nil {
		return cfg, err
	}
	if cfg.BidTTL, err = envDuration("DSPSIM_BID_TTL", 3600, time.Second); err != nil {
		return cfg, err
	}
	if cfg.MaxBids, err = envInt("DSPSIM_MAX_BIDS", 10_000); err != nil {
		return cfg, err
	}
	if cfg.Seed, err = envInt64("DSPSIM_SEED", 0); err != nil {
		return cfg, err
	}
	cfg.StrictDemand = envString("DSPSIM_STRICT_DEMAND", "") == "true"

	return cfg, cfg.Validate()
}

// Validate rejects configurations that would produce nonsensical bids.
func (c Config) Validate() error {
	if c.PriceMultMin <= 0 || c.PriceMultMax <= 0 {
		return fmt.Errorf("price multipliers must be positive, got min=%v max=%v", c.PriceMultMin, c.PriceMultMax)
	}
	if c.PriceMultMin > c.PriceMultMax {
		return fmt.Errorf("DSPSIM_PRICE_MULT_MIN (%v) must not exceed DSPSIM_PRICE_MULT_MAX (%v)", c.PriceMultMin, c.PriceMultMax)
	}
	if c.NoBidRate < 0 || c.NoBidRate > 1 {
		return fmt.Errorf("DSPSIM_NO_BID_RATE must be in [0,1], got %v", c.NoBidRate)
	}
	if c.FallbackFloor <= 0 {
		return fmt.Errorf("DSPSIM_FALLBACK_FLOOR must be positive, got %v", c.FallbackFloor)
	}
	if c.MaxPrice <= 0 {
		return fmt.Errorf("DSPSIM_MAX_PRICE must be positive, got %v", c.MaxPrice)
	}
	if c.MaxBids <= 0 {
		return fmt.Errorf("DSPSIM_MAX_BIDS must be positive, got %v", c.MaxBids)
	}
	return nil
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def, fmt.Errorf("%s: %v", key, err)
	}
	return f, nil
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("%s: %v", key, err)
	}
	return i, nil
}

func envInt64(key string, def int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def, fmt.Errorf("%s: %v", key, err)
	}
	return i, nil
}

// envDuration reads a bare number and multiplies it by unit, so that
// DSPSIM_LATENCY_MS=250 means 250ms and DSPSIM_CATALOG_TTL=60 means 60s.
func envDuration(key string, def int64, unit time.Duration) (time.Duration, error) {
	n, err := envInt64(key, def)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("%s must not be negative, got %d", key, n)
	}
	return time.Duration(n) * unit, nil
}
