package telemetry

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestOutcomeFromDemand(t *testing.T) {
	parseErr := errors.New("invalid bid json")
	netErr := errors.New("connection reset")

	tests := []struct {
		name   string
		err    error
		isBid  bool
		status int
		want   string
	}{
		{name: "timeout", err: context.DeadlineExceeded, status: 0, want: OutcomeTimeout},
		{name: "timeout wrapped", err: errors.Join(errors.New("adapter"), context.DeadlineExceeded), status: 200, want: OutcomeTimeout},
		{name: "http 4xx", err: errors.New("unauthorized"), status: http.StatusUnauthorized, want: OutcomeHTTPError},
		{name: "http 5xx", err: nil, status: http.StatusBadGateway, want: OutcomeHTTPError},
		{name: "status 0 with err", err: netErr, status: 0, want: OutcomeHTTPError},
		{name: "malformed parse", err: parseErr, status: http.StatusOK, want: OutcomeMalformed},
		{name: "bid", isBid: true, status: http.StatusOK, want: OutcomeBid},
		{name: "nobid 204", status: http.StatusNoContent, want: OutcomeNoBid},
		{name: "nobid empty seat", status: http.StatusOK, want: OutcomeNoBid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OutcomeFromDemand(tt.err, tt.isBid, tt.status)
			if got != tt.want {
				t.Errorf("OutcomeFromDemand() = %q, want %q", got, tt.want)
			}
		})
	}
}
