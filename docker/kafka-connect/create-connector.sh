#!/bin/bash

set -e

echo "Kafka Connect is ready. Recreating S3 Sink Connector..."

curl -v -X DELETE http://localhost:8083/connectors/s3-sink || true

curl -v -X POST http://localhost:8083/connectors \
  -H "Content-Type: application/json" \
  --data "@/scripts/connector-config.json"
