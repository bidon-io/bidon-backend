package adapters_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/bidding/rendering"
)

func TestEnrichBid_parsesExtWhenRenderingNil(t *testing.T) {
	ext := json.RawMessage(`{"rendering":{"creative":{"type":"vast"}}}`)
	dr := &adapters.DemandResponse{
		DemandID: adapter.AdikteevKey,
		Bid: &adapters.NormalizedBid{
			DemandID: adapter.AdikteevKey,
			Ext:      ext,
		},
	}

	adapters.EnrichBid(dr)

	require.NotNil(t, dr.Bid.Rendering)
	assert.Equal(t, rendering.CreativeTypeVAST, dr.Bid.Rendering.Creative.Type)
}

func TestEnrichBid_nilExtAppliesDefaults(t *testing.T) {
	dr := &adapters.DemandResponse{
		DemandID: adapter.AmazonKey,
		Bid:      &adapters.NormalizedBid{DemandID: adapter.AmazonKey},
	}

	adapters.EnrichBid(dr)

	require.NotNil(t, dr.Bid.Rendering)
	assert.Equal(t, rendering.CreativeTypeStaticImage, dr.Bid.Rendering.Creative.Type)
	assert.Equal(t, rendering.CloseButtonStyleIconX, dr.Bid.Rendering.CloseButton.Style)
}

func TestEnrichBid_skipsWhenRenderingAlreadySet(t *testing.T) {
	existing := &rendering.Config{
		Creative: &rendering.CreativeConfig{Type: rendering.CreativeTypeMRAID},
	}
	dr := &adapters.DemandResponse{
		DemandID: adapter.VungleKey,
		Bid: &adapters.NormalizedBid{
			Ext:       json.RawMessage(`{"rendering":{"creative":{"type":"vast"}}}`),
			Rendering: existing,
		},
	}

	adapters.EnrichBid(dr)

	assert.Same(t, existing, dr.Bid.Rendering)
	assert.Equal(t, rendering.CreativeTypeMRAID, dr.Bid.Rendering.Creative.Type)
}

func TestEnrichBid_noBidIsNoop(t *testing.T) {
	adapters.EnrichBid(nil)
	adapters.EnrichBid(&adapters.DemandResponse{})
}
