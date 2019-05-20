//go:generate go build github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. filesystem/behavior/probe_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. rsync/engine.proto rsync/receive.proto rsync/transmission.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/daemon/daemon.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/prompt/prompt.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/session/session.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. session/configuration.proto session/scan_mode.proto session/selection.proto session/session.proto session/stage_mode.proto session/state.proto session/version.proto session/watch_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. session/endpoint/remote/endpoint_protocol.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. sync/archive.proto sync/cache.proto sync/change.proto sync/conflict.proto sync/entry.proto sync/ignore_vcs_mode.proto sync/mode.proto sync/problem.proto sync/symlink_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. url/url.proto
//go:generate rm ./protoc-gen-go

package pkg
