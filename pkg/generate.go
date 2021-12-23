//go:build generate

//go:generate go build google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go build google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative filesystem/behavior/probe_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative forwarding/configuration.proto forwarding/session.proto forwarding/socket_overwrite_mode.proto forwarding/state.proto forwarding/version.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative forwarding/endpoint/remote/protocol.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative selection/selection.proto
//go:generate protoc --plugin=./protoc-gen-go --plugin=./protoc-gen-go-grpc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/daemon/daemon.proto
//go:generate protoc --plugin=./protoc-gen-go --plugin=./protoc-gen-go-grpc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/forwarding/forwarding.proto
//go:generate protoc --plugin=./protoc-gen-go --plugin=./protoc-gen-go-grpc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/prompting/prompting.proto
//go:generate protoc --plugin=./protoc-gen-go --plugin=./protoc-gen-go-grpc -I. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative service/synchronization/synchronization.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative synchronization/configuration.proto synchronization/scan_mode.proto synchronization/session.proto synchronization/stage_mode.proto synchronization/state.proto synchronization/version.proto synchronization/watch_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative synchronization/core/archive.proto synchronization/core/cache.proto synchronization/core/change.proto synchronization/core/conflict.proto synchronization/core/entry.proto synchronization/core/ignore_vcs_mode.proto synchronization/core/mode.proto synchronization/core/problem.proto synchronization/core/symbolic_link_mode.proto synchronization/core/ignorer_mode.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative synchronization/endpoint/remote/protocol.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative synchronization/rsync/engine.proto synchronization/rsync/receive.proto synchronization/rsync/transmission.proto
//go:generate protoc --plugin=./protoc-gen-go -I. --go_out=. --go_opt=paths=source_relative url/url.proto
//go:generate rm ./protoc-gen-go ./protoc-gen-go-grpc

package pkg

import (
	// HACK: For some reason, the google.golang.org/grpc/cmd/protoc-gen-go-grpc
	// command is actually a separate Go module, so Go complains that it's not
	// covered by our go.mod/go.sum even if google.golang.org/grpc is present
	// there. Thus, we use these ghost imports just to get go mod tidy to pick
	// up on these dependencies and keep them in go.mod/go.sum. We don't really
	// need it for google.golang.org/protobuf/cmd/protoc-gen-go since it's not
	// a separate module from google.golang.org/protobuf, but it's best to make
	// things as future-proof as possible. This file also makes a great location
	// for doing these imports since it couples these imports conceptually with
	// the commands above and it isn't included in any part of the build.
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
