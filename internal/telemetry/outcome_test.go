package telemetry

import (
	"context"
	"errors"
	"net/http"
	"testing"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
)

func TestOutcomeFromDemand(t *testing.T) {
	parseErr := errors.New("invalid bid json")
	netErr := errors.New("connection reset")

	tests := []struct {
		name   string
		err    error
		isBid  bool
		status int
		want   telemetryv1.Outcome
	}{
		{name: "timeout", err: context.DeadlineExceeded, status: 0, want: telemetryv1.Outcome_OUTCOME_TIMEOUT},
		{name: "timeout wrapped", err: errors.Join(errors.New("adapter"), context.DeadlineExceeded), status: 200, want: telemetryv1.Outcome_OUTCOME_TIMEOUT},
		{name: "http 4xx", err: errors.New("unauthorized"), status: http.StatusUnauthorized, want: telemetryv1.Outcome_OUTCOME_HTTP_ERROR},
		{name: "http 5xx", err: nil, status: http.StatusBadGateway, want: telemetryv1.Outcome_OUTCOME_HTTP_ERROR},
		{name: "status 0 with err", err: netErr, status: 0, want: telemetryv1.Outcome_OUTCOME_HTTP_ERROR},
		{name: "malformed parse", err: parseErr, status: http.StatusOK, want: telemetryv1.Outcome_OUTCOME_MALFORMED},
		{name: "bid", isBid: true, status: http.StatusOK, want: telemetryv1.Outcome_OUTCOME_BID},
		{name: "nobid 204", status: http.StatusNoContent, want: telemetryv1.Outcome_OUTCOME_NOBID},
		{name: "nobid empty seat", status: http.StatusOK, want: telemetryv1.Outcome_OUTCOME_NOBID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OutcomeFromDemand(tt.err, tt.isBid, tt.status)
			if got != tt.want {
				t.Errorf("OutcomeFromDemand() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutcomeLabels pins the dsp_response_total outcome labels to the enum,
// so a new Outcome value gets a label and dashboards see renames as failures.
func TestOutcomeLabels(t *testing.T) {
	want := map[telemetryv1.Outcome]string{
		telemetryv1.Outcome_OUTCOME_UNSPECIFIED: "unspecified",
		telemetryv1.Outcome_OUTCOME_BID:         "bid",
		telemetryv1.Outcome_OUTCOME_NOBID:       "nobid",
		telemetryv1.Outcome_OUTCOME_TIMEOUT:     "timeout",
		telemetryv1.Outcome_OUTCOME_HTTP_ERROR:  "http_error",
		telemetryv1.Outcome_OUTCOME_MALFORMED:   "malformed",
	}
	if len(want) != len(telemetryv1.Outcome_name) {
		t.Fatalf("Outcome enum has %d values, test pins %d", len(telemetryv1.Outcome_name), len(want))
	}
	for o, label := range want {
		if got := outcomeLabel(o); got != label {
			t.Errorf("outcomeLabel(%v) = %q, want %q", o, got, label)
		}
	}
}
