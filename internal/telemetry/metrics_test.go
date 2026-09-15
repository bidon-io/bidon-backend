package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestObserveDSPAndAuction(t *testing.T) {
	dsp := "bac61_metrics_dsp"
	ObserveDSP(dsp, OutcomeBid, 0.12)
	ObserveDSP(dsp, OutcomeTimeout, 4.0)
	ObserveAuctionCompleted(AuctionResultOK)
	ObserveAuctionCompleted(AuctionResultError)

	bid, err := DSPResponseTotal.GetMetricWithLabelValues(dsp, OutcomeBid)
	if err != nil {
		t.Fatalf("dsp_response_total bid: %v", err)
	}
	if got := counterValue(t, bid); got < 1 {
		t.Errorf("dsp_response_total{outcome=bid} = %v, want >= 1", got)
	}

	timeout, err := DSPResponseTotal.GetMetricWithLabelValues(dsp, OutcomeTimeout)
	if err != nil {
		t.Fatalf("dsp_response_total timeout: %v", err)
	}
	if got := counterValue(t, timeout); got < 1 {
		t.Errorf("dsp_response_total{outcome=timeout} = %v, want >= 1", got)
	}

	hist, ok := DSPRequestDuration.WithLabelValues(dsp).(prometheus.Metric)
	if !ok {
		t.Fatal("dsp_request_duration_seconds is not prometheus.Metric")
	}
	if got := histogramCount(t, hist); got < 2 {
		t.Errorf("dsp_request_duration_seconds sample_count = %v, want >= 2", got)
	}

	completed, err := AuctionCompletedTotal.GetMetricWithLabelValues(AuctionResultOK)
	if err != nil {
		t.Fatalf("auction_completed_total ok: %v", err)
	}
	if got := counterValue(t, completed); got < 1 {
		t.Errorf("auction_completed_total{result=ok} = %v, want >= 1", got)
	}
}

func counterValue(t *testing.T, m interface{ Write(*dto.Metric) error }) float64 {
	t.Helper()
	pb := &dto.Metric{}
	if err := m.Write(pb); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	return pb.GetCounter().GetValue()
}

func histogramCount(t *testing.T, m interface{ Write(*dto.Metric) error }) uint64 {
	t.Helper()
	pb := &dto.Metric{}
	if err := m.Write(pb); err != nil {
		t.Fatalf("write metric: %v", err)
	}
	return pb.GetHistogram().GetSampleCount()
}
