//go:generate go build github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. daemon/service/daemon.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. filesystem/watch.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. prompt/service/prompt.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. rsync/receive.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. session/configuration.proto session/session.proto session/state.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. session/service/session.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. sync/archive.proto sync/cache.proto sync/change.proto sync/conflict.proto sync/entry.proto sync/ignore.proto sync/problem.proto sync/symlink.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. url/url.proto
//go:generate rm ./protoc-gen-go

package pkg
