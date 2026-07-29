# Onboard a new network

This guide covers every change needed to onboard a new demand source / RTB bidding adapter.

The changes fall into five layers:

| Layer                                    | Purpose                                         |
|------------------------------------------|-------------------------------------------------|
| **Adapter key**                          | Declare the network identity (`adapter.Key`)    |
| **Bidding adapter + builder**            | OpenRTB request/response handling               |
| **SDK init config**                      | Configuration sent to the mobile SDK at startup |
| **Test factories + test expectations**   | Database test fixtures                          |
| **Seeds, UI constants, HTTP test files** | Sample data, admin UI, and API test helpers     |

See [the implementation details](network_impl.md) for a step by step guide.


### Questions

- do they have protobuf spec for bid request?
- how to map sdk request ad types to ad formats?
- how to deliver the creative, are we proxying?
- how to render ad - creative rendering configuration (close/container/endcard/type)
