# ADR-0004: Standalone Redis

**Status:** Accepted
**Date:** 2026-09-07
**Linear:** [BAC-55](https://linear.app/bidon/issue/BAC-55/replace-devstagingprod-redis-cluster-with-standalone-redis)

---

## Context

The Go client was constructed as a Redis Cluster client everywhere. Dev compose tried to run a 3-node cluster. The cluster bus port (client port + 10000) was not published, so every node stayed in `cluster_state:fail` and clients failed with "Could not initialize cluster slots cache."

Staging and production were a second, independent bug. Both passed a `redis://` URL into `ClusterOptions.Addrs`, which expects `host:port`, and pointed it at a Redis that was not cluster-enabled. `ClusterClient` connects lazily, so the process booted and only failed on the first cache read.

Bidon uses Redis as a cache (bid cache, auction results). It does not shard a dataset that outgrows one node.

## Decision

Dev, staging, and the deployed Redis are standalone. `config.NewRedisClient` reads `REDIS_URL` and builds a `redis.UniversalClient`: one address is a standalone client, several comma-separated addresses are a cluster client. `NewRedisClient` pings Redis before returning, so any service that builds a client refuses to boot if the ping fails.

Rejected: repair the 3-node dev cluster (publish the bus port, fix announce addresses, keep the `nodes.conf` volume). The cluster was not buying anything the cache needed, and staging and production were not clustered either.

## Consequences

- `REDIS_CLUSTER` is gone. A deployment that still sets it, and not `REDIS_URL`, will fail at boot instead of silently using a stale value.
- Moving an environment to a real cluster later is a config change (more than one address). Call sites depend on `redis.UniversalClient`, not `*redis.ClusterClient`.
- `docker-compose-prod.yml` was left on its existing Redis image. The client and the URL are what changed there.
