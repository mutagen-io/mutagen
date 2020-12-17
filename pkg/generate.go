// +build generate

//go:generate go build github.com/golang/protobuf/protoc-gen-go
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. filesystem/behavior/probe_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. forwarding/configuration.proto forwarding/session.proto forwarding/socket_overwrite_mode.proto forwarding/state.proto forwarding/version.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. forwarding/endpoint/remote/protocol.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. selection/selection.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/daemon/daemon.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/forwarding/forwarding.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/prompting/prompting.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative,plugins=grpc:. service/synchronization/synchronization.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. synchronization/configuration.proto synchronization/scan_mode.proto synchronization/session.proto synchronization/stage_mode.proto synchronization/state.proto synchronization/version.proto synchronization/watch_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. synchronization/core/archive.proto synchronization/core/cache.proto synchronization/core/change.proto synchronization/core/conflict.proto synchronization/core/entry.proto synchronization/core/ignore_vcs_mode.proto synchronization/core/mode.proto synchronization/core/problem.proto synchronization/core/symlink_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. synchronization/endpoint/remote/protocol.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. synchronization/rsync/engine.proto synchronization/rsync/receive.proto synchronization/rsync/transmission.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=paths=source_relative:. url/url.proto
//go:generate rm ./protoc-gen-go

package pkg
