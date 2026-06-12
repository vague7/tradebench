#!/bin/sh
apk add --no-cache protoc protobuf-dev
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:/go/bin

OUT=./shared/proto/gen
mkdir -p $OUT
protoc \
  --go_out=$OUT --go_opt=paths=source_relative \
  --go-grpc_out=$OUT --go-grpc_opt=paths=source_relative \
  -I ./shared/proto \
  ./shared/proto/telemetry.proto \
  ./shared/proto/sandbox.proto \
  ./shared/proto/bot_fleet.proto
