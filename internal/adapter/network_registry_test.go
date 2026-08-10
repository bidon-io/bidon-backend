package adapter_test

import (
	"testing"

	"github.com/bidon-io/bidon-backend/internal/adapter"
)

func TestNetworks_CoverAllKeys(t *testing.T) {
	t.Parallel()

	registered := make(map[adapter.Key]struct{}, len(adapter.Networks()))
	for _, n := range adapter.Networks() {
		if n.Key == "" {
			t.Fatal("network with empty key")
		}
		if n.Label == "" {
			t.Fatalf("%s: empty label", n.Key)
		}
		if n.AccountType == "" {
			t.Fatalf("%s: empty account type", n.Key)
		}
		if _, dup := registered[n.Key]; dup {
			t.Fatalf("duplicate network key %q", n.Key)
		}
		registered[n.Key] = struct{}{}
	}

	for _, key := range adapter.Keys {
		if _, ok := registered[key]; !ok {
			t.Fatalf("adapter.Keys entry %q missing from network registry", key)
		}
	}
	if len(registered) != len(adapter.Keys) {
		t.Fatalf("registry size %d != adapter.Keys size %d", len(registered), len(adapter.Keys))
	}
}

func TestNetworkByKey(t *testing.T) {
	t.Parallel()

	n, ok := adapter.NetworkByKey(adapter.MolocoKey)
	if !ok {
		t.Fatal("expected Moloco in registry")
	}
	if n.Label != "Moloco" || n.AccountType != "DemandSourceAccount::Moloco" {
		t.Fatalf("unexpected moloco entry: %+v", n)
	}
	if !n.SupportsBidding || n.SupportsWaterfall {
		t.Fatalf("moloco bidding/waterfall flags = %v/%v", n.SupportsBidding, n.SupportsWaterfall)
	}
	if n.EnvSecret != adapter.EnvSecretMoloco {
		t.Fatalf("EnvSecret = %v, want Moloco", n.EnvSecret)
	}

	if _, ok := adapter.NetworkByKey(adapter.Key("missing")); ok {
		t.Fatal("expected missing key lookup to fail")
	}
}

func TestBiddingAndWaterfallKeys(t *testing.T) {
	t.Parallel()

	bidding := map[adapter.Key]struct{}{}
	for _, key := range adapter.BiddingKeys() {
		bidding[key] = struct{}{}
		n, ok := adapter.NetworkByKey(key)
		if !ok || !n.SupportsBidding {
			t.Fatalf("BiddingKeys contains %q without SupportsBidding", key)
		}
	}
	waterfall := map[adapter.Key]struct{}{}
	for _, key := range adapter.WaterfallKeys() {
		waterfall[key] = struct{}{}
		n, ok := adapter.NetworkByKey(key)
		if !ok || !n.SupportsWaterfall {
			t.Fatalf("WaterfallKeys contains %q without SupportsWaterfall", key)
		}
	}

	if _, ok := bidding[adapter.AmazonKey]; !ok {
		t.Fatal("amazon should be bidding")
	}
	if _, ok := waterfall[adapter.AdmobKey]; !ok {
		t.Fatal("admob should be waterfall")
	}
	if _, ok := bidding[adapter.BidmachineKey]; !ok {
		t.Fatal("bidmachine should be bidding")
	}
	if _, ok := waterfall[adapter.BidmachineKey]; !ok {
		t.Fatal("bidmachine should be waterfall")
	}
}

func TestApplyProcessedConfig(t *testing.T) {
	t.Parallel()

	n, ok := adapter.NetworkByKey(adapter.MintegralKey)
	if !ok {
		t.Fatal("mintegral missing")
	}

	dest := map[string]any{}
	n.ApplyProcessedConfig(
		dest,
		map[string]any{"publisher_id": "pub-1"},
		map[string]any{"app_id": "app-1"},
		nil,
	)
	if dest["seller_id"] != "pub-1" || dest["app_id"] != "app-1" {
		t.Fatalf("without ad unit: %+v", dest)
	}
	if _, ok := dest["tag_id"]; ok {
		t.Fatalf("tag_id should be omitted without ad unit: %+v", dest)
	}

	dest = map[string]any{}
	n.ApplyProcessedConfig(
		dest,
		map[string]any{"publisher_id": "pub-1"},
		map[string]any{"app_id": "app-1"},
		map[string]any{"unit_id": "unit-1", "placement_id": "plc-1"},
	)
	if dest["tag_id"] != "unit-1" || dest["placement_id"] != "plc-1" {
		t.Fatalf("with ad unit: %+v", dest)
	}
}
