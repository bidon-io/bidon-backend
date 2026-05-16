# RTB

Supports RTB 2.3 and 2.5.

# Onboard a new network

- RTB adapter integration: 
  - bid requests
  - parses responses
  - manages DSP connections
  - delivers rendering configs
  - and fires callbacks
- add it to the admin

Questions:
- how to deliver the creative, are we proxying? 
- how to render ad - creative rendering configuration (close/container/endcard/type)
- protobuf spec?


# uSDK General flow

1. /config - get all adapter configs
2. SDK collects tokens for adapters based on config
3. /aution - SDK sends tokens to server
4. Server sends S2S to Networks with tokens
5. Server conducts auction with network responses
6. Server builds auction response with mix of bidding and CPM units
7. SDK continues auction on client side trying to get fill from most expensive to cheapest one by one
8. After finishing auction SDK sends /stats to server
9. Server sends win/loss to Bidding networks based on /stats request

Data that SDK sends to /auction:
- auction_id - can be passed manually by publisher, or default one can be picked up
- banner.format = BANNER - publisher can load any ad type that he wants

