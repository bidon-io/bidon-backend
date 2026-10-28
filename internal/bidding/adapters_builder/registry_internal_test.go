package adapters_builder

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

func TestBiddingAdapters_RegisteredAsBiddingNetworks(t *testing.T) {
	t.Parallel()

	for key := range biddingAdapters {
		n, ok := adapter.NetworkByKey(key)
		if !ok {
			t.Errorf("bidding adapter %q missing from network registry", key)
			continue
		}
		if !n.SupportsBidding {
			t.Errorf("bidding adapter %q registered without SupportsBidding", key)
		}
	}
}
