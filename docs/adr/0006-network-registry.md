# ADR-0006: Single network registry

**Status:** Accepted
**Date:** 2026-10-29
**Linear:** [BAC-45](https://linear.app/bidon/issue/BAC-45/single-network-registry-across-runtime-admin-and-ui)

---

## Context

Adding a demand network meant editing the same facts in several places: the Go adapter key, the bidding-config switch that copied or renamed `extra` fields, env-secret injection, admin account-type validators, and two UI maps (`Networks.js` and `DemandSourceOptions.js`). Those copies drifted. The UI file already called itself the canonical list, and the backend did not read it.

## Decision

`internal/adapter` holds the network catalog: key, label, account type, whether the network bids or waterfalls, and the `CopyKey` / `RenameKey` projections for account, app, and ad-unit extras. `AdaptersConfigBuilder` reads that catalog instead of a per-network switch. Admin exposes it as `GET /api/networks`. The admin UI dropdowns load from that endpoint. The static UI maps are gone.

The onboarding checklist is [the network runbook](../runbooks/network.md).

Rejected: keep `Networks.js` as the source of truth and sync Go from it by hand. Also rejected, for this change: generating the per-network OpenAPI and Yup extra forms from the registry. Those forms stay declarative. Amazon stays on its own bidding path; the registry still describes it.

## Consequences

- A new network is still an adapter, an OpenAPI key, seeds, and any custom extra forms. Membership, labels, and the standard field renames are one registry entry.
- The UI cannot show a network the API catalog does not return. A backend image newer than `bidon-ui` is fine for the dropdown; a UI that still has a hardcoded list is the old bug.
- OpenRTB request and response helpers are a separate change. This registry does not replace them.
