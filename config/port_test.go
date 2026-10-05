package config_test

import (
	"testing"

	"github.com/bidon-io/bidon-backend/config"
)

func TestHTTPPort(t *testing.T) {
	tests := []struct {
		name       string
		serviceEnv string
		port       string
		want       string
	}{
		{name: "service env wins over PORT", serviceEnv: "1324", port: "1323", want: "1324"},
		{name: "falls back to PORT", port: "3100", want: "3100"},
		{name: "falls back to default", want: "1324"},
		{name: "blank values are ignored", serviceEnv: " ", port: " ", want: "1324"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SDKAPI_PORT", tt.serviceEnv)
			t.Setenv("PORT", tt.port)

			if got := config.HTTPPort("SDKAPI_PORT", "1324"); got != tt.want {
				t.Errorf("HTTPPort() = %q, want %q", got, tt.want)
			}
		})
	}
}
