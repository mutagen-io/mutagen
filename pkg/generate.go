//go:generate go build github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. filesystem/behavior/probe_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. selection/selection.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/daemon/daemon.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/prompt/prompt.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/synchronization/synchronization.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. synchronization/configuration.proto synchronization/scan_mode.proto synchronization/session.proto synchronization/stage_mode.proto synchronization/state.proto synchronization/version.proto synchronization/watch_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. synchronization/core/archive.proto synchronization/core/cache.proto synchronization/core/change.proto synchronization/core/conflict.proto synchronization/core/entry.proto synchronization/core/ignore_vcs_mode.proto synchronization/core/mode.proto synchronization/core/problem.proto synchronization/core/symlink_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. synchronization/endpoint/remote/protocol.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. synchronization/rsync/engine.proto synchronization/rsync/receive.proto synchronization/rsync/transmission.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. url/url.proto
//go:generate rm ./protoc-gen-go

package pkg
