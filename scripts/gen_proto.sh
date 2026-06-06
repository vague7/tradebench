#!/bin/bash
set -e
OUT=./shared/proto/gen
mkdir -p $OUT
protoc \
  --go_out=$OUT --go_opt=paths=source_relative \
  --go-grpc_out=$OUT --go-grpc_opt=paths=source_relative \
  -I ./shared/proto \
  ./shared/proto/telemetry.proto \
  ./shared/proto/sandbox.proto \
  ./shared/proto/bot_fleet.proto
echo "Proto generation complete."
