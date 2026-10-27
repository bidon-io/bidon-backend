package telemetry

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

// NoticeMeta is the envelope identity threaded onto a notice send.
// AppID is required for joins on (app_id, auction_id). Zero means the caller
// did not have it; /v2/show always sets it.
type NoticeMeta struct {
	AppID     int64
	SessionID string
	Country   string
	AdType    string
	AdFormat  string
	// Base is a copy of the request that triggered the notice. HasBase is
	// false when the caller had no request to copy.
	Base    schema.BaseRequest
	HasBase bool
	present bool
}

// Present reports whether WithNoticeMeta stored this value.
func (m NoticeMeta) Present() bool {
	return m.present
}

type noticeMetaKey struct{}

// WithNoticeMeta attaches notice envelope identity to ctx.
func WithNoticeMeta(ctx context.Context, meta NoticeMeta) context.Context {
	meta.present = true
	return context.WithValue(ctx, noticeMetaKey{}, meta)
}

// NoticeMetaFrom returns the identity attached by WithNoticeMeta, or the zero value.
func NoticeMetaFrom(ctx context.Context) NoticeMeta {
	meta, _ := ctx.Value(noticeMetaKey{}).(NoticeMeta)
	return meta
}
