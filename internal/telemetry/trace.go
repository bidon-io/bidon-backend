package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const TracerName = "bidon-sdkapi/auction"

const (
	SpanAuctionRun = "auction.run"
	SpanAuctionDSP = "auction.dsp"
)

const (
	AttrAppID     = "app_id"
	AttrAuctionID = "auction_id"
	AttrDSP       = "dsp"
	AttrOutcome   = "outcome"
)

func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}

func (p Params) withContextTrace(ctx context.Context) Params {
	if p.TraceID == "" {
		p.TraceID = TraceIDFromContext(ctx)
	}
	return p
}
