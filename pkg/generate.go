//go:generate go build github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. daemon/service/daemon.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. filesystem/watch.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. prompt/service/prompt.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. rsync/receive.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. session/configuration.proto session/session.proto session/state.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. session/service/session.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. sync/archive.proto sync/cache.proto sync/change.proto sync/conflict.proto sync/entry.proto sync/ignore.proto sync/problem.proto sync/symlink.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=plugins=grpc:. url/url.proto
//go:generate rm ./protoc-gen-go

package pkg
