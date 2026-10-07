package config

import (
	"os"
	"strings"
)

// HTTPPort returns the HTTP listen port for a service: the service-specific
// env var (e.g. SDKAPI_PORT) first, then the shared PORT, then fallback.
// PORT stays as a fallback so deployments that set it (staging, prod) keep working.
func HTTPPort(serviceEnv, fallback string) string {
	for _, key := range []string{serviceEnv, "PORT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return fallback
}
