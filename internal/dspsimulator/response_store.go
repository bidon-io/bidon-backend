package dspsimulator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
)

type ResponseStore struct {
	responses map[string][]*openrtb2.BidResponse
}

func NewResponseStore(dir string) (*ResponseStore, error) {
	store := &ResponseStore{
		responses: make(map[string][]*openrtb2.BidResponse),
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read response dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_bidres.json") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", entry.Name(), err)
		}

		var resp openrtb2.BidResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		key := deriveKey(entry.Name(), &resp)
		store.responses[key] = append(store.responses[key], &resp)
	}

	return store, nil
}

func (s *ResponseStore) Lookup(key string) *openrtb2.BidResponse {
	candidates, ok := s.responses[key]
	if !ok || len(candidates) == 0 {
		return nil
	}
	idx := rand.Intn(len(candidates))
	return candidates[idx]
}

var descriptorTable = map[string]struct {
	mediaType string
	variant   string
	dims      bool
}{
	"banner":             {mediaType: "banner", variant: "", dims: true},
	"banner_mrec":        {mediaType: "banner", variant: "", dims: true},
	"interstitial":       {mediaType: "banner", variant: "", dims: true},
	"interstitial_mraid": {mediaType: "banner", variant: "mraid", dims: true},
	"rewarded_vast":      {mediaType: "video", variant: "vast", dims: false},
	"rewarded_native":    {mediaType: "native", variant: "", dims: false},
}

func deriveKey(filename string, resp *openrtb2.BidResponse) string {
	base := strings.TrimSuffix(filename, "_bidres.json")
	parts := strings.SplitN(base, "_", 3)
	if len(parts) < 3 {
		return filename
	}

	os := parts[0]
	dsp := parts[1]
	descriptor := parts[2]

	entry, ok := descriptorTable[descriptor]
	if !ok {
		return filename
	}

	key := dsp + "/" + os + "/" + entry.mediaType
	if entry.dims {
		w, h := extractDims(resp)
		key += "/" + fmt.Sprintf("%dx%d", w, h)
	}
	if entry.variant != "" {
		key += "/" + entry.variant
	}

	return key
}

func extractDims(resp *openrtb2.BidResponse) (int64, int64) {
	if len(resp.SeatBid) == 0 || len(resp.SeatBid[0].Bid) == 0 {
		return 0, 0
	}
	bid := resp.SeatBid[0].Bid[0]
	return bid.W, bid.H
}
