# Onboard a new network

Those are the steps to onboard a new network:

- RTB adapter integration:
    - new adapter in `internal/adapter/adapter.go`
    - adapter implementation in `internal/bidding/adapters/` implementing:
        - bid requests
        - parses responses
        - manages DSP connections
        - delivers rendering configs
        - and fires callbacks
- add it as a new constant to the Admin UI in those files located under `web/bidon_ui/constants/`
    - `DemandSourceOptions.js`
    - `Networks.js`
- create sample seed data (app, auctions, line items)

Questions:

- how to deliver the creative, are we proxying?
- hot to map sdk request ad types to ad formats?
- how to render ad - creative rendering configuration (close/container/endcard/type)
- protobuf spec for bid request?
