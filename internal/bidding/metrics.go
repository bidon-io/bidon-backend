package bidding

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AdapterOptionsErrors counts ExecuteOptions failures, labelled by adapter.
// It keeps climbing after the once-per-app Error log, so a misconfiguration
// stays visible in metrics for as long as auctions keep hitting it.
var AdapterOptionsErrors = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "adapter_options_errors_total",
	Help: "Auctions whose adapter ExecuteOptions failed.",
}, []string{"adapter"})
