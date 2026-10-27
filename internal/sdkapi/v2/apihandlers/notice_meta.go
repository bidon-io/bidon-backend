package apihandlers

import (
	"context"

	"github.com/bidon-io/bidon-backend/internal/sdkapi"
	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
	"github.com/bidon-io/bidon-backend/internal/telemetry"
)

func withNoticeMeta(ctx context.Context, app sdkapi.App, base schema.BaseRequest, country, adType, adFormat string) context.Context {
	return telemetry.WithNoticeMeta(ctx, telemetry.NoticeMeta{
		AppID:     app.ID,
		SessionID: base.Session.ID,
		Country:   country,
		AdType:    adType,
		AdFormat:  adFormat,
	})
}
