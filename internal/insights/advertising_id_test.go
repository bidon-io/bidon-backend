package insights

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/sdkapi/schema"
)

func TestResolveAdvertisingID(t *testing.T) {
	tests := []struct {
		name string
		req  InitRequest
		want string
	}{
		{
			name: "missing base request",
			req: InitRequest{
				IDFA: "idfa",
				IDG:  "idg",
				IDFV: "idfv",
			},
			want: "",
		},
		{
			name: "ios prefers idfa",
			req: InitRequest{
				IDFA: "idfa",
				IDFV: "idfv",
				BaseRequest: &schema.BaseRequest{
					Device: schema.Device{OS: "iOS"},
				},
			},
			want: "idfa",
		},
		{
			name: "ios falls back to idfv",
			req: InitRequest{
				IDFV: "idfv",
				BaseRequest: &schema.BaseRequest{
					Device: schema.Device{OS: "iOS"},
				},
			},
			want: "idfv",
		},
		{
			name: "android uses idg",
			req: InitRequest{
				IDG: "idg",
				BaseRequest: &schema.BaseRequest{
					Device: schema.Device{OS: "android"},
				},
			},
			want: "idg",
		},
		{
			name: "unknown os",
			req: InitRequest{
				IDFA: "idfa",
				IDG:  "idg",
				IDFV: "idfv",
				BaseRequest: &schema.BaseRequest{
					Device: schema.Device{OS: "unknown"},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveAdvertisingID(tt.req)
			if got != tt.want {
				t.Fatalf("ResolveAdvertisingID() = %q, want %q", got, tt.want)
			}
		})
	}
}
