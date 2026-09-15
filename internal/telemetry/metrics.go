package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	DSPResponseTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dsp_response_total",
		Help: "Bidding-round DSP outcomes.",
	}, []string{"dsp", "outcome"})

	DSPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dsp_request_duration_seconds",
		Help:    "DSP send-to-return latency.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 4, 8},
	}, []string{"dsp"})

	AuctionCompletedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auction_completed_total",
		Help: "Auction Run completions.",
	}, []string{"result"})
)

func ObserveDSP(dsp, outcome string, seconds float64) {
	DSPResponseTotal.WithLabelValues(dsp, outcome).Inc()
	DSPRequestDuration.WithLabelValues(dsp).Observe(seconds)
}

func ObserveAuctionCompleted(result string) {
	AuctionCompletedTotal.WithLabelValues(result).Inc()
}
