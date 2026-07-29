# How to Add a New Network to BidOn

## Step-by-Step

### 1. Register the Adapter Key

**File:** `internal/adapter/adapter.go`

Add a new constant (sorted alphabetically in the `const` block) and append it to
the `Keys` slice:

```go
SmadexKey Key = "smadex"   // add in const block
SmadexKey,                  // add to Keys slice
```

### 2. Create the Bidding Adapter

**New directory:** `internal/bidding/adapters/<key>/`

**New files:**
- `<key>.go` — the adapter implementation
- `<key>_test.go` — unit tests

The adapter must:

- Implement `CreateRequest`, `ExecuteRequest`, and `ParseBids` (the three methods
  required by the `adapters.Adapter` interface).
- Define a `Builder` function that returns `*adapters.Bidder`.
- Handle three ad types: `BannerType`, `InterstitialType`, `RewardedType`.
- Set `DisplayManager` and `DisplayManagerVer` on each impression.
- Marshal `App.Ext` with `sdk_instance_id` (pulled from `auctionRequest.AdObject.Demands`).

Tests cover `CreateRequest` (Banner BANNER, Banner MREC, Interstitial, Rewarded),
`ExecuteRequest`, `ParseBids` (200, 204, 4xx), and `Builder`.

Copy an existing adapter (e.g. `adikteev/adikteev.go`) and update:
- Package name
- Type name (`SmadexAdapter`)
- Adapter key constant (`adapter.SmadexKey`)
- Endpoint URL
- The `Builder` comment

### 3. Register the Adapter in the Builder

**File:** `internal/bidding/adapters_builder/adapters_builder.go`

Three changes:
1. Add import for the new adapter package.
2. Add `adapter.SmadexKey: smadex.Builder` to the `biddingAdapters` map.
3. Add a `case` in the `Build` config builder for your key (the Smadex adapter
   copies the Adikteev pattern: `adaptersMap[key]["sdk_instance_id"]
   = extra["sdk_instance_id"]`).

### 4. Create the SDK Init Config

**File:** `internal/sdkapi/adapter_init_config.go`

Add a new struct implementing `AdapterInitConfig`:

```go
type SmadexInitConfig struct {
    SdkInstanceID string `json:"sdk_instance_id"`
    Order         int    `json:"order"`
}

func (a *SmadexInitConfig) Key() adapter.Key { return adapter.SmadexKey }
func (a *SmadexInitConfig) SetDefaultOrder()  { a.Order = 0 }
```

Then add a `case adapter.SmadexKey: config = new(SmadexInitConfig)` in
`NewAdapterInitConfig`.

### 5. Update Database Test Factories

Two files need a new `case` for the adapter key:

**`internal/db/dbtest/demand_source_account_factory.go`**
```go
case adapter.SmadexKey:
    return []byte(`{}`)
```

**`internal/db/dbtest/app_demand_profile_factory.go`**
```go
case adapter.SmadexKey:
    return []byte(`{"sdk_instance_id": "sdk_instance_id"}`)
```

### 6. Update Store Test Expectations

**File:** `internal/sdkapi/store/store_test.go`

The `TestAdapterInitConfigsFetcher_FetchAdapterInitConfigs_Valid` test iterates
over `adapter.Keys` and creates profiles for every adapter. Add the expected
`SmadexInitConfig` entry in the `want` slice for the "second app" group (both
`setOrder: false` and `setOrder: true` test cases):

```go
&sdkapi.SmadexInitConfig{
    SdkInstanceID: "sdk_instance_id",
    Order:         0,
},
```

### 7. Add Demand Source Seed

**File:** `cmd/bidon-seed/seeds/20250325114723_demand_sources.sql`

```sql
('smadex', 'Smadex', NOW(), NOW())
```

### 8. Add Sample App Data

**File:** `cmd/bidon-seed/sample_seeds/20260323121000_sample_apps_and_configurations.sql`

This is the largest change. Within the single PL/pgSQL `DO $$ ... END $$` block:

1. **Declare variables:** `smadex_id BIGINT`, a unique `snake_app_id BIGINT
   := 2006`, and `smadex_account_id BIGINT := 3006`.
2. **Resolve the demand source:** `SELECT id INTO smadex_id FROM demand_sources
   WHERE api_key = 'smadex'` + a null-check `IF smadex_id IS NULL THEN RAISE
   EXCEPTION ...`.
3. **Insert a demand source account:**
   ```sql
   (smadex_account_id, smadex_id, owner_id,
    'DemandSourceAccount::smadex', '{}'::jsonb,
    true, false, NOW(), NOW(), 'Smadex Audience Network', smadex_account_id)
   ```
4. **Insert the demo app:**
   ```sql
   (snake_app_id, owner_id, 1, 'Snake', 'com.demo.snake',
    'snake_' || snake_app_id, '{}'::jsonb, NOW(), NOW(), snake_app_id)
   ```
5. **Insert an AppDemandProfile:**
   ```sql
   (4013, snake_app_id, 'DemandSourceAccount::Smadex', smadex_account_id, smadex_id,
    '{"sdk_instance_id": "sdk_instance_id"}'::jsonb, NOW(), NOW(), 4013, true)
   ```
6. **Insert line items** (Banner, MREC, Interstitial, Rewarded) — one per format.
   Assign unique IDs (e.g. 5401–5404).
7. **Insert auction configurations** (Banner, Interstitial, Rewarded) referencing
   the line items above. Set `ad_type` to the correct integer, `demands` and
   `bidding` arrays to `ARRAY['smadex']::varchar[]`, and `ad_unit_ids` to the
   relevant line item IDs.

### 9. Update Admin UI Constants

**`web/bidon_ui/constants/DemandSourceOptions.js`** — Add the option entry:
```js
{ label: "Smadex", value: "DemandSourceAccount::Smadex" },
```

**`web/bidon_ui/constants/Networks.js`** — Add to `NETWORK_DEFS`:
```js
{ key: "smadex", label: "Smadex", accountType: "DemandSourceAccount::Smadex" },
```
And add `"smadex"` to the `BIDDING_NETWORK_KEYS` array.

### 10. Create HTTP Test File & Testdata

**HTTP test file:** `sdk-<key>.http`
Contains one Config request + 8 auction requests (iOS/Android ×
Banner/MREC/Interstitial/Rewarded), each pointing to a JSON testdata file.

**Testdata files:** 9 JSON files under
`internal/sdkapi/v2/apihandlers/testdata/`:
- `config/<key>_config_request.json` — config endpoint payload
- `auction/<key>/ios_<key>_banner.json`
- `auction/<key>/ios_<key>_banner_mrec.json`
- `auction/<key>/ios_<key>_interstitial.json`
- `auction/<key>/ios_<key>_rewarded.json`
- `auction/<key>/android_<key>_banner.json`
- `auction/<key>/android_<key>_banner_mrec.json`
- `auction/<key>/android_<key>_interstitial.json`
- `auction/<key>/android_<key>_rewarded.json`

Each auction JSON references the demo app (`com.demo.snake` / `snake_2006`) and
the Smadex adapter with a dummy `token`. Each config JSON also references the
same app bundle/key and adapter.

---

## Verification Checklist

- [ ] `go build ./cmd/bidon-admin/...` compiles
- [ ] `go build ./cmd/bidon-sdkapi/...` compiles
- [ ] `go build ./cmd/bidon-seed/...` compiles
- [ ] `go test ./internal/bidding/adapters/<key>/...` passes
- [ ] `go vet ./internal/bidding/adapters/<key>/...` clean
- [ ] `go vet ./internal/adapter/...` clean
- [ ] `go vet ./internal/bidding/adapters_builder/...` clean
- [ ] `go vet ./internal/sdkapi/...` clean
- [ ] Store integration tests pass (may require clean test DB): `go test
  ./internal/sdkapi/store/...`

---

## Checklist Summary

| # | File | Action |
|---|------|--------|
| 1 | `internal/adapter/adapter.go` | Add `Key` constant + add to `Keys` |
| 2 | `internal/bidding/adapters/<key>/<key>.go` | New: adapter implementation |
| 3 | `internal/bidding/adapters/<key>/<key>_test.go` | New: unit tests |
| 4 | `internal/bidding/adapters_builder/adapters_builder.go` | Import + map + config case |
| 5 | `internal/sdkapi/adapter_init_config.go` | New struct + switch case |
| 6 | `internal/db/dbtest/demand_source_account_factory.go` | Add case for extra |
| 7 | `internal/db/dbtest/app_demand_profile_factory.go` | Add case for data |
| 8 | `internal/sdkapi/store/store_test.go` | Add expected config in want slices |
| 9 | `cmd/bidon-seed/seeds/20250325114723_demand_sources.sql` | Add demand source row |
| 10 | `cmd/bidon-seed/sample_seeds/20260323121000_sample_apps_and_configurations.sql` | Account + app + profile + line items + auctions |
| 11 | `web/bidon_ui/constants/DemandSourceOptions.js` | Add option entry |
| 12 | `web/bidon_ui/constants/Networks.js` | Add to NETWORK_DEFS + BIDDING_NETWORK_KEYS |
| 13 | `sdk-<key>.http` | New: Config + 8 auction HTTP requests |
| 14 | `internal/sdkapi/v2/apihandlers/testdata/config/<key>_config_request.json` | New |
| 15–22 | `internal/sdkapi/v2/apihandlers/testdata/auction/<key>/{ios,android}_<key>_{banner,banner_mrec,interstitial,rewarded}.json` | New: 8 files |
