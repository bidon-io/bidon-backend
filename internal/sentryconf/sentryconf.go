// Package sentryconf provides sentry configuration for Bidon services
package sentryconf

import (
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var ClientOptions = sentry.ClientOptions{
	Dsn:              os.Getenv("SENTRY_DSN"),
	Debug:            env != "production",
	AttachStacktrace: true,
	EnableTracing:    true,
	TracesSampleRate: 1.0,
	SendDefaultPII:   true,
	Environment:      env,
}
var FlushTimeout = 2 * time.Second

var env = os.Getenv("ENVIRONMENT")
