package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	telemetryv1 "github.com/bidon-io/bidon-backend/pkg/proto/org/bidon/telemetry/v1"
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

	TelemetryDroppedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "telemetry_dropped_total",
		Help: "Catalog events not delivered to telemetry-events.",
	}, []string{"reason"})

	// NoticeDeliveryTotal type is nurl|lurl|burl|timeout. result is 2xx|4xx|5xx|timeout.
	NoticeDeliveryTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "notice_delivery_total",
		Help: "OpenRTB notice attempts. type is nurl, lurl, burl, or timeout (TimeoutURL). result is 2xx, 4xx, 5xx, or timeout (transport failure).",
	}, []string{"type", "result"})

	AdImpressionTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ad_impression_total",
		Help: "ad_impression rows emitted, synthesized or client.",
	})
)

const (
	DropReasonInvalid    = "invalid"
	DropReasonMarshal    = "marshal"
	DropReasonBufferFull = "buffer_full"
	DropReasonProduce    = "produce_error"
)

func ObserveDSP(dsp string, outcome telemetryv1.Outcome, seconds float64) {
	DSPResponseTotal.WithLabelValues(dsp, outcomeLabel(outcome)).Inc()
	DSPRequestDuration.WithLabelValues(dsp).Observe(seconds)
}

func ObserveAuctionCompleted(result AuctionResult) {
	AuctionCompletedTotal.WithLabelValues(string(result)).Inc()
}

func ObserveNotice(kind, result string) {
	NoticeDeliveryTotal.WithLabelValues(kind, result).Inc()
}

func ObserveAdImpression() {
	AdImpressionTotal.Inc()
}
