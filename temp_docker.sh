
apk add --no-cache protobuf-dev dos2unix
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:/go/bin
dos2unix ./scripts/gen_proto.sh
chmod +x ./scripts/gen_proto.sh
./scripts/gen_proto.sh
