package segment

import (
	"encoding/json"
	"github.com/bidon-io/bidon-backend/internal/admin"
	"github.com/bidon-io/bidon-backend/internal/db"
)

type SegmentExt struct {
	Gender            string                 `json:"gender"`
	TotalInAppsAmount int                    `json:"total_in_apps_amount"`
	IsPaying          bool                   `json:"is_paying"`
	GameLevel         int                    `json:"game_level"`
	Age               int                    `json:"age"`
	CustomAttributes  map[string]interface{} `json:"custom_attributes"`
}

type Params struct {
	Country string `json:"country"`
	Ext     string `json:"ext"`
}

func Match(sgmnts []db.Segment, params *Params) *db.Segment {
	for _, sgmnt := range sgmnts {
		isMatched := false

		for _, filter := range sgmnt.Filters {
			switch filter.Type {
			case "country":
				isMatched = matchCountry(filter, params.Country)
			case "custom_string":
				isMatched = matchCustomString(filter, params.Ext)
			}

			if isMatched {
				return &sgmnt
			}
		}
	}

	return nil
}

func matchCountry(filter admin.SegmentFilter, country string) bool {
	switch filter.Operator {
	case "IN":
		return containsString(filter.Values, country)
	case "NOT IN":
		return !containsString(filter.Values, country)
	default:
		return false
	}
}

func matchCustomString(filter admin.SegmentFilter, Ext string) bool {
	var parsedExt SegmentExt

	if err := json.Unmarshal([]byte(Ext), &parsedExt); err != nil {
		return false
	}

	switch filter.Operator {
	case "==":
		return filter.Values[0] == parsedExt.CustomAttributes[filter.Name]
	case "!=":
		return filter.Values[0] != parsedExt.CustomAttributes[filter.Name]
	default:
		return false
	}
}

func containsString(values []string, str string) bool {
	for _, v := range values {
		if v == str {
			return true
		}
	}
	return false
}
