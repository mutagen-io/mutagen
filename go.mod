module github.com/mutagen-io/mutagen

go 1.22.0
toolchain go1.24.1

require (
	github.com/Microsoft/go-winio v0.6.2
	github.com/bmatcuk/doublestar/v4 v4.7.1
	github.com/dustin/go-humanize v1.0.1
	github.com/eknkc/basex v1.0.1
	github.com/fatih/color v1.17.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/google/uuid v1.6.0
	github.com/hectane/go-acl v0.0.0-20230122075934-ca0b05cb1adb
	github.com/klauspost/compress v1.17.11
	github.com/mattn/go-isatty v0.0.20
	github.com/mutagen-io/extstat v0.0.0-20210224131814-32fa3f057fa8
	github.com/mutagen-io/fsevents v0.0.0-20230629001834-f53e17b91ebc
	github.com/mutagen-io/gopass v0.0.0-20230214181532-d4b7cdfe054c
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/zeebo/xxh3 v1.0.2
	golang.org/x/net v0.38.0
	golang.org/x/sys v0.31.0
	golang.org/x/text v0.23.0
	google.golang.org/grpc v1.67.1
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1
	google.golang.org/protobuf v1.35.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.3
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	golang.org/x/term v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
)

replace k8s.io/apimachinery v0.21.3 => github.com/mutagen-io/apimachinery v0.21.3-mutagen1
