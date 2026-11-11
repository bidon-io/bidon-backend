# Onboard a new network

How to add a demand network (DSP or SDK network) end to end: registry, adapter,
admin, SDK config, tests, seeds, and a local check against `bidon-dspsim`.

The canonical registration is the Go network registry. Admin UI network lists
and account-type dropdowns come from `GET /api/networks`, so there is no
separate UI catalog to update.

The worked example throughout is Smadex, added in
[#934](https://github.com/bidon-io/bidon-backend/pull/934) (BAC-25).

## Before you start

Get answers to these from the network before writing code:

- **Bidding or waterfall?** Bidding networks need an RTB adapter. Waterfall-only
  networks stop after the SDK init config.
- **Which fields, and at which level?** Account (`demand_source_accounts.extra`),
  app (`app_demand_profiles.data`), or ad unit (`line_items.extra`).
- **Where do secrets live?** In the account `extra`, or in env (the Meta and
  Moloco pattern).
- **How is the creative delivered?** Proxy or DSP URL.
- **How do SDK ad types map to ad formats?** Banner, MREC, interstitial,
  rewarded, and the sizes for each.
- **What belongs in the Bidon rendering config?** Close, container, endcard,
  type. See the [rendering schema](../architecture/rendered-configuration-schema.md).
- **Is there a bid-request contract beyond OpenRTB?** For example protobuf.

## 1. Register the network

1. **Adapter key.** Add the `Key` constant in `internal/adapter/adapter.go` and
   append it to `Keys`.
2. **Registry entry.** Add a `Network` to `networks` in
   `internal/adapter/network_registry.go`:
   - `Key`, `Label`, `AccountType` (`DemandSourceAccount::<Name>`)
   - `SupportsBidding` / `SupportsWaterfall`
   - Field projections: `AccountExtra`, `AppData` and `AdUnitExtra`, using
     `CopyKey` (same name) or `RenameKey`. Networks with no projections pass
     their account `extra` through to the processed config unchanged.
   - Optional `InjectEnvSecrets` when secrets come from env.
3. **Projection test.** Add a case to `remapParityCases` in
   `internal/bidding/adapters_builder/adapters_config_builder_test.go`.
4. **OpenAPI adapter key.** Add the key to
   `internal/admin/openapi/schemas/adapter-key.schema.json` and run
   `go generate ./internal/admin/api/...`.

Smadex needs only its account `endpoint` in the processed config:

```go
{
	Key:             SmadexKey,
	Label:           "Smadex",
	AccountType:     "DemandSourceAccount::Smadex",
	SupportsBidding: true,
	AccountExtra:    []FieldMap{CopyKey("endpoint")},
},
```

## 2. Bidding adapter

Bidding networks only.

1. Create `internal/bidding/adapters/<network>/` with a type that implements
   `adapters.BidderInterface` (`internal/bidding/adapters/bidder.go`):
   - `BuildImpression` returns the OpenRTB `Imp` for the ad type, plus request
     options.
   - `ExecuteOptions` returns the URL and headers.
2. Add optional behaviour by implementing the extra interfaces that the shared
   path type-asserts: `OpenRTBRequestEnricher`, `OpenRTBBidEnricher`,
   `CustomBidParser`, `CustomRequestBuilder`. Smadex uses only
   `EnrichOpenRTBRequest`, to put the SDK token in `app.ext`.
3. Write a `Builder(adapter.ProcessedConfigsMap, *http.Client)` that reads the
   processed config. Fall back to a default endpoint when the account has none.
4. Register the builder in `biddingAdapters`
   (`internal/bidding/adapters_builder/adapters_builder.go`).

Use the shared execute and parse path. Bid parsing happens at the call site, so
an adapter does not need its own `ParseBids` unless the response is not
OpenRTB. Proprietary flows, such as Amazon `FetchBids`, are wired beside the
registry.

## 3. SDK init config

Add a case to `NewAdapterInitConfig` and an `<Name>InitConfig` struct in
`internal/sdkapi/adapter_init_config.go`. This is what `/v2/config` returns to
the SDK for the network. Check `internal/sdkapi/store/` for special cases.

## 4. Admin validation and UI

1. **API validation.** Add a case to `extraRule` in
   `internal/admin/demand_source_account.go` for the account `extra` fields.
   Without a case, the API accepts any `extra`.
2. **Line item and app extras.** Add the network's schemas under
   `internal/admin/openapi/schemas/` and validators when it has ad-unit or app
   fields.
3. **UI form.** Add the fields and a Yup schema in
   `web/bidon_ui/components/demand_source_accounts/DemandSourceAccountExtraFormFields.vue`
   when the network has account fields.

## 5. Tests

- **Adapter.** `internal/bidding/adapters/<network>/<network>_test.go`: one
  case per ad type, plus the builder and the endpoint fallback.
- **Factories.** Add cases to `ValidDemandSourceAccountExtra`
  (`internal/db/dbtest/demand_source_account_factory.go`) and
  `ValidAppDemandProfileData` (`internal/db/dbtest/app_demand_profile_factory.go`).
- **Init config.** Add the expected config to
  `TestAdapterInitConfigsFetcher_FetchAdapterInitConfigs_Valid` in
  `internal/sdkapi/store/store_test.go`.
- **Request fixtures.** Add auction requests per platform and ad type under
  `internal/sdkapi/v2/apihandlers/testdata/auction/<network>/`, and a config
  request under `testdata/config/`.

## 6. Seeds

`just seed` runs `bidon-seed -reset -sample -dev`, which applies three folders
in `cmd/bidon-seed/`:

| Folder | Runs | Add |
|---|---|---|
| `seeds/` | always | The `demand_sources` row for the network. |
| `sample_seeds/` | `-sample` (local, staging, previews) | A sample account with the real DSP endpoint, app demand profiles, and line items or auction configs that include the network. |
| `dev_sample_seeds/` | `-dev` (local only) | An overlay that points the account `endpoint` at `http://bidon-dspsim:1325/openrtb/bid`. |

Keep the real URL in `sample_seeds/`, so staging and previews call the real
DSP. See `dev_sample_seeds/20260323121200_smadex_dspsim_endpoint.sql` for the
overlay pattern.

## 7. Verify locally

1. Start the stack and reseed:

   ```bash
   just compose
   just seed
   ```

2. Check that the network is listed at `GET /api/networks` and that you can
   create an account for it in the admin UI (<http://localhost:3010>).
3. Add `sdk-<network>.http` at the repository root, with a config request and
   one auction request per fixture. Copy `sdk-smadex.http`.
4. Send the config request. The network's init config is in the response.
5. Send an auction request. The response ranks a bid from the network.
6. Open <http://localhost:1325/debug/bids> to see the bid `bidon-dspsim`
   answered, and the nurl, burl and lurl it advertises.
7. Run the tests for the touched packages:

   ```bash
   go test ./internal/adapter/... ./internal/bidding/... ./internal/sdkapi/... ./internal/admin/...
   ```

See [ADR 0001](../adr/0001-dsp-simulator.md) for how the simulator works.

## Smadex walkthrough

The files #934 touched, in the order of this runbook:

| Step | File |
|---|---|
| Key | `internal/adapter/adapter.go` |
| Registry | `internal/adapter/network_registry.go` |
| Projection test | `internal/bidding/adapters_builder/adapters_config_builder_test.go` |
| OpenAPI key | `internal/admin/openapi/schemas/adapter-key.schema.json`, `internal/admin/api/api.gen.go` |
| Adapter | `internal/bidding/adapters/smadex/smadex.go`, `smadex_test.go` |
| Builder | `internal/bidding/adapters_builder/adapters_builder.go` |
| Init config | `internal/sdkapi/adapter_init_config.go`, `internal/sdkapi/store/store_test.go` |
| Factories | `internal/db/dbtest/demand_source_account_factory.go`, `app_demand_profile_factory.go` |
| Fixtures | `internal/sdkapi/v2/apihandlers/testdata/auction/smadex/`, `testdata/config/smadex_config_request.json` |
| Seeds | `seeds/20250325114723_demand_sources.sql`, `sample_seeds/20260323121000_sample_apps_and_configurations.sql`, `dev_sample_seeds/20260323121200_smadex_dspsim_endpoint.sql` |
| Manual requests | `sdk-smadex.http` |

Smadex has no admin `extraRule` case or UI form fields yet (step 4), so its
`endpoint` is only editable through the API without validation.
