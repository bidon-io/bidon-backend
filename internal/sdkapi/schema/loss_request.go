package schema

import "github.com/shopspring/decimal"

type LossRequest struct {
	ShowRequest
	ExternalWinner ExternalWinner `json:"external_winner"`
}

type ExternalWinner struct {
	DemandID string           `json:"demand_id"`
	ECPM     *decimal.Decimal `json:"ecpm"` // Deprecated: ECPM is deprecated since 0.7, use Price instead
	Price    *decimal.Decimal `json:"price"`
}

func (e *ExternalWinner) GetPrice() decimal.Decimal {
	if e.Price != nil {
		return *e.Price
	}

	if e.ECPM != nil {
		return *e.ECPM
	}

	return decimal.Zero
}
