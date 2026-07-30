package adapters_test

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/ad"
	"github.com/bidon-io/bidon-backend/internal/bidding/adapters"
)

func TestResolveBannerSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		format     ad.Format
		isAdaptive bool
		isTablet   bool
		opts       adapters.BannerSizeOptions
		wantW      int64
		wantH      int64
		wantErr    bool
	}{
		{
			name:   "banner",
			format: ad.BannerFormat,
			wantW:  320,
			wantH:  50,
		},
		{
			name:   "leaderboard",
			format: ad.LeaderboardFormat,
			wantW:  728,
			wantH:  90,
		},
		{
			name:   "mrec",
			format: ad.MRECFormat,
			wantW:  300,
			wantH:  250,
		},
		{
			name:   "adaptive phone",
			format: ad.AdaptiveFormat,
			wantW:  320,
			wantH:  50,
		},
		{
			name:       "adaptive tablet upgrades to leaderboard",
			format:     ad.AdaptiveFormat,
			isAdaptive: true,
			isTablet:   true,
			wantW:      728,
			wantH:      90,
		},
		{
			name:       "adaptive tablet rejected when opt-out set",
			format:     ad.AdaptiveFormat,
			isAdaptive: true,
			isTablet:   true,
			opts:       adapters.BannerSizeOptions{RejectAdaptiveLeaderboard: true},
			wantErr:    true,
		},
		{
			name:   "empty format defaults to banner size",
			format: ad.EmptyFormat,
			wantW:  320,
			wantH:  50,
		},
		{
			name:   "unknown format falls back to empty default",
			format: ad.Format("UNKNOWN"),
			wantW:  320,
			wantH:  50,
		},
		{
			name:       "non-adaptive tablet keeps format size",
			format:     ad.BannerFormat,
			isAdaptive: false,
			isTablet:   true,
			wantW:      320,
			wantH:      50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := adapters.ResolveBannerSize(tt.format, tt.isAdaptive, tt.isTablet, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got[0] != tt.wantW || got[1] != tt.wantH {
				t.Fatalf("size = %dx%d, want %dx%d", got[0], got[1], tt.wantW, tt.wantH)
			}
		})
	}
}

func TestBannerFormats_CommonKeys(t *testing.T) {
	t.Parallel()

	required := []ad.Format{
		ad.BannerFormat,
		ad.LeaderboardFormat,
		ad.MRECFormat,
		ad.AdaptiveFormat,
		ad.EmptyFormat,
	}
	for _, format := range required {
		if _, ok := adapters.BannerFormats[format]; !ok {
			t.Fatalf("BannerFormats missing %q", format)
		}
	}
}
