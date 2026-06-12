import subprocess

script = """
apk add --no-cache protobuf-dev dos2unix
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:/go/bin
dos2unix ./scripts/gen_proto.sh
chmod +x ./scripts/gen_proto.sh
./scripts/gen_proto.sh
"""

with open("temp_docker.sh", "w", newline="\n") as f:
    f.write(script)

subprocess.run(["docker", "run", "--rm", "-v", r"C:\Users\win11\Desktop\tradebench:/app", "-w", "/app", "golang:1.25-alpine", "sh", "temp_docker.sh"])
