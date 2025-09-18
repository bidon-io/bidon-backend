package amazon

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/bidon-io/bidon-backend/internal/adapter"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

type Slot struct {
	SlotUUID   string `json:"slot_uuid"`
	PricePoint string `json:"price_point"`
}

type PricePointsMap map[string]PricePoint

type PricePoint struct {
	Price      decimal.Decimal `json:"price"`
	PricePoint string          `json:"price_point"`
}

type Adapter struct {
	PricePointsMap PricePointsMap
}

func (a *Adapter) FetchBids(auctionRequest *schema.AuctionRequest) ([]*adapters.DemandResponse, error) {
	slotsJSON, ok := auctionRequest.AdObject.Demands[adapter.AmazonKey]["token"].(string)
	if !ok {
		return nil, fmt.Errorf("no token in request")
	}
	var slots []Slot
	err := json.Unmarshal([]byte(slotsJSON), &slots)
	if err != nil {
		return nil, err
	}

	demandResponses := make([]*adapters.DemandResponse, 0, len(slots))
	for _, slot := range slots {
		if pricePoint, ok := a.PricePointsMap[slot.PricePoint]; ok {
			ID, _ := uuid.NewV4()
			impID, _ := uuid.NewV4()
			demandResponse := adapters.DemandResponse{
				DemandID: adapter.AmazonKey,
				SlotUUID: slot.SlotUUID,
				Bid: &adapters.BidDemandResponse{
					DemandID: adapter.AmazonKey,
					ID:       ID.String(),
					ImpID:    impID.String(),
					Price:    pricePoint.Price,
				},
			}
			demandResponses = append(demandResponses, &demandResponse)
		} else {
			demandResponse := adapters.DemandResponse{
				DemandID: adapter.AmazonKey,
				Error:    fmt.Errorf("cannot find price point"),
				SlotUUID: slot.SlotUUID,
			}
			demandResponses = append(demandResponses, &demandResponse)
		}
	}

	return demandResponses, nil
}

func Builder(cfg adapter.ProcessedConfigsMap) (*Adapter, error) {
	amazonCfg := cfg[adapter.AmazonKey]

	pricePointsMapRaw, ok := amazonCfg["price_points_map"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("price_points_map is not set")
	}

	pricePointsMap := make(PricePointsMap)
	for key, value := range pricePointsMapRaw {
		var point PricePoint
		if v, ok := value.(map[string]any); ok {
			switch price := v["price"].(type) {
			case json.Number:
				point.Price, _ = decimal.NewFromString(price.String())
			case decimal.Decimal:
				point.Price = price
			case float64:
				point.Price = decimal.NewFromFloat(price)
			default:
				return nil, fmt.Errorf("invalid price type for price_point: %s", key)
			}

			if pricePoint, ok := v["price_point"].(string); ok {
				point.PricePoint = pricePoint
			}
			pricePointsMap[key] = point
		} else {
			return nil, fmt.Errorf("invalid structure for price_point: %s", key)
		}
	}

	adpt := &Adapter{
		PricePointsMap: pricePointsMap,
	}

	return adpt, nil
}
