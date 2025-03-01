# Runner Controller Microservice
Go implementation of the Runner Controller.

## Setup

1. Install Go
2. Install the protobuf-grpc compiler.
```sh
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```
4. Run the following command to start the server.
```sh
go run main.go
```

# Editing .proto files

1. Install protoc compiler
2. Run the following command to generate the go files from the proto files.
```sh
protoc --go_out=./ --go_opt=paths=source_relative \
    --go-grpc_out=./ --go-grpc_opt=paths=source_relative \
    ./proto/authenticate.proto
```
