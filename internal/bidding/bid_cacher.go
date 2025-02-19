package bidding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/benbjohnson/clock"
	"time"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/redis/go-redis/v9"
)

const TTL = 5 * 60 * time.Second

type BidCache struct {
	Redis *redis.ClusterClient
	Clock clock.Clock
}

func (c *Cache) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Cache) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}

// Cache -> Struct to hold the cache data in Redis
type Cache struct {
	Bids map[adapter.Key]CacheEntry // Store latest bid for each adapter
}

type CacheEntry struct {
	Bid       adapters.DemandResponse
	CreatedAt time.Time
	AuctionID string // AuctionID for which the bid was made, it will help to send notifications in future
}

// ApplyBidCache gets the auction result, stores it in the cache and enhances response with the cache data if available
// The cache key is generated based on the session ID and the ad type
// TTL is set to 5 minutes
func (b *BidCache) ApplyBidCache(ctx context.Context, br *schema.BiddingRequest, result *AuctionResult) []adapters.DemandResponse {
	if _, ok := br.GetExtData()["bid_cache"]; !ok { // If the request has bid_cache field, only then cache the bids
		return result.Bids
	}
	cacheKey := fmt.Sprintf("bidding:%s:%s", br.Session.ID, br.AdType)
	inCache := &Cache{Bids: make(map[adapter.Key]CacheEntry)}
	err := b.Redis.GetDel(ctx, cacheKey).Scan(inCache)
	if err != nil && !errors.Is(err, redis.Nil) { // Some error occurred while fetching the cache
		fmt.Printf("Error fetching bid cache: %v\n", err)
		return result.Bids
	}

	// Remove expired cache entries
	for key, entry := range inCache.Bids {
		if b.Clock.Since(entry.CreatedAt) > TTL {
			delete(inCache.Bids, key)
			// TODO: Send LURL notification here
		}
	}

	// If the cache is empty and the result is empty, do nothing
	if len(inCache.Bids) == 0 && len(result.Bids) == 0 {
		return []adapters.DemandResponse{}
	}

	// Split the bids into cache and response data
	toResponse, toCache := splitBids(result.Bids)

	// Select the highest bid for each adapter
	now := b.Clock.Now()
	for _, bid := range toCache {
		cacheEntry := CacheEntry{Bid: bid, CreatedAt: now, AuctionID: br.Session.ID}
		if existing, ok := inCache.Bids[bid.DemandID]; ok {
			if bid.Price() > existing.Bid.Price() {
				inCache.Bids[bid.DemandID] = cacheEntry
				// TODO: Send LURL notification here for the existing bid
			}
		} else {
			inCache.Bids[bid.DemandID] = cacheEntry
		}
	}

	// Get the highest bid from cache and put it in the response
	demand, bid := getMax(&inCache.Bids)
	delete(inCache.Bids, demand)
	toResponse = append(toResponse, bid.Bid)

	// Write the rest cache back to Redis if not empty
	if len(inCache.Bids) > 0 {
		bytes, _ := inCache.MarshalBinary()
		err = b.Redis.Set(ctx, cacheKey, string(bytes), TTL).Err()
		if err != nil {
			fmt.Printf("Error writing bid cache: %v\n", err)
		}
	}

	return toResponse
}

// splitBids splits the given bids into two slices: one for bids that can be cached and one for bids that should be included in the response.
// It returns two slices: toResponse and toCache.
func splitBids(bids []adapters.DemandResponse) (toResponse, toCache []adapters.DemandResponse) {
	for _, bid := range bids {
		if bid.CanCache() {
			toCache = append(toCache, bid)
		} else {
			toResponse = append(toResponse, bid)
		}
	}
	return
}

// getMaxAndRemove get max bid from map and remove it, mutates the map
func getMax(m *map[adapter.Key]CacheEntry) (adapter.Key, CacheEntry) {
	var maxDemand adapter.Key
	var maxValue CacheEntry
	first := true

	for demand, entry := range *m {
		if first || entry.Bid.Price() > maxValue.Bid.Price() {
			maxDemand = demand
			maxValue = entry
			first = false
		}
	}

	return maxDemand, maxValue
}
