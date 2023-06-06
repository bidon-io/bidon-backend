package sgmnt

import "github.com/bidon-io/bidon-backend/internal/admin"

type Params struct {
	Country string `json:"country"`
}

func Match(segments []admin.Segment, params Params) *admin.Segment {
	for _, segment := range segments {
		isMatched := false

		for _, filter := range segment.Filters {
			switch filter.Type {
			case "country":
				isMatched = matchCountry(filter, params.Country)
			}
		}

		if isMatched {
			return &segment
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

func containsString(values []string, str string) bool {
	for _, v := range values {
		if v == str {
			return true
		}
	}
	return false
}
