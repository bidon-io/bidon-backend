# Bidon backend docs

Index for this repository. Local setup and staging still live in the
[repository README](../README.md).

## How a request moves

The SDK calls sdkapi (`/v2/config`, then `/v2/auction`). The auction matches a
segment, loads config, builds demand from line items and bidding adapters, and
runs a parallel bidding round. Ranked ad units go back to the SDK, which
reports win, show, and loss.

A DSP can attach a rendering config at `seatbid.bid.ext.rendering`. The auction
exposes that block at `ext.rendering`, and the uSDK renderer uses it for the
close button, endcards, and creative container. Auction and DSP metrics for
this path are in the telemetry runbook.

Diagrams:

- [uSDK sequence](architecture/uSDK_flow.puml) — config, auction, win/show/loss
- [RTB sequence](architecture/RTB_flow.puml) — bid request through notification
- [Admin domain model](architecture/admin_domain_model.puml)

## Layout

| Directory | What belongs here |
| --- | --- |
| [architecture/](architecture/) | Domain model, flows, and schemas |
| [adr/](adr/) | Architecture decision records. Numbering is sequential and never reused. |
| [runbooks/](runbooks/) | How to operate a service or onboard a network |
| [dev/](dev/) | Contributor tooling |
| [skills/](skills/) | Agent skills for DSP-facing config |
| [archive/backlog/](archive/backlog/) | Superseded plans and reading copies |

## Architecture

- [Ad types and formats](architecture/ad.md)
- [Rendering configuration schema](architecture/rendered-configuration-schema.md)
- [uSDK sequence](architecture/uSDK_flow.puml)
- [RTB sequence](architecture/RTB_flow.puml)
- [Admin domain model](architecture/admin_domain_model.puml)

## ADRs

- [Index](adr/README.md)
- [0001 — DSP simulator](adr/0001-dsp-simulator.md)

## Runbooks

- [Onboard a network](runbooks/network.md)
- [Telemetry POC: local metrics](runbooks/telemetry-poc.md) — local VictoriaMetrics queries
- [Redpanda Console and vmui](runbooks/telemetry-console-and-vmui.pdf) — preview 957 walkthrough, including `POST /v2/telemetry`

## Dev

- [Stacked PRs with git-spice](dev/git-spice.md)

## Skills

- [Rendering configuration](skills/bidon_render_configuration/SKILL.md) — field
  rules, format pairings, and worked payloads in the same directory

## Archive

The September telemetry plan and the warehouse TRD draft. Current how-tos are
the [local metrics note](runbooks/telemetry-poc.md) and the
[Console and vmui guide](runbooks/telemetry-console-and-vmui.pdf). See
[archive/backlog/](archive/backlog/).
