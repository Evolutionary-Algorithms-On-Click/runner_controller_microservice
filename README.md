# Runner Controller Microservice
Go implementation of the Runner Controller.

## Setup

### Prerequisites

1. GoLang - Install GoLang.
2. MinIO - Start a MinIO server instance.
3. CockroachDB - Start a CockroachDB instance.

### Installation

1. Install the protobuf-grpc compiler.
```sh
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```
2. Export the following environment variables.
```sh
export DATABASE_URL=<cockroachdb_url>
export MINIO_ENDPOINT=<minio_endpoint>
export MINIO_ACCESS_KEY_ID=<minio_access_key>
export MINIO_SECRET_KEY=<minio_secret_key>
```
3. Run the following command to start the server.
```sh
go run main.go
```

### Editing `.proto` files

1. Install protoc compiler
2. Run the following command to generate the go files from the proto files.
```sh
protoc --go_out=./ --go_opt=paths=source_relative \
    --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
    ./proto/authenticate.proto
```
