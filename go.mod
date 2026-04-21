module github.com/mutagen-io/mutagen

go 1.25.0

require (
	github.com/Microsoft/go-winio v0.6.2
	github.com/bmatcuk/doublestar/v4 v4.7.1
	github.com/dustin/go-humanize v1.0.1
	github.com/eknkc/basex v1.0.1
	github.com/fatih/color v1.19.0
	github.com/fsnotify/fsevents v0.2.0
	github.com/google/uuid v1.6.0
	github.com/hectane/go-acl v0.0.0-20230122075934-ca0b05cb1adb
	github.com/klauspost/compress v1.18.5
	github.com/mattn/go-isatty v0.0.21
	github.com/mutagen-io/extstat v0.0.0-20210224131814-32fa3f057fa8
	github.com/mutagen-io/gopass v0.0.0-20230214181532-d4b7cdfe054c
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/zeebo/xxh3 v1.1.0
	go.yaml.in/yaml/v4 v4.0.0-rc.4
	golang.org/x/net v0.53.0
	golang.org/x/sys v0.43.0
	golang.org/x/text v0.36.0
	google.golang.org/grpc v1.80.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.1
	google.golang.org/protobuf v1.36.11
	k8s.io/apimachinery v0.21.3
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	golang.org/x/term v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
)

replace k8s.io/apimachinery v0.21.3 => github.com/mutagen-io/apimachinery v0.21.3-mutagen1
