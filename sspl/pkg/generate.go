//go:build generate

//go:generate go build google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go build google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative licensing/state.proto
//go:generate protoc --plugin=./protoc-gen-go --plugin=./protoc-gen-go-grpc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/licensing/licensing.proto
//go:generate rm ./protoc-gen-go ./protoc-gen-go-grpc

package pkg

// NOTE: We are also reliant on the side-effect imports in pkg/generate.go, but
// we don't duplicate them here because they only need to exist in one place.
