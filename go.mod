module github.com/mutagen-io/mutagen

go 1.19

require (
	github.com/Microsoft/go-winio v0.5.2
	github.com/bmatcuk/doublestar/v4 v4.2.0
	github.com/dustin/go-humanize v1.0.0
	github.com/eknkc/basex v1.0.1
	github.com/fatih/color v1.13.0
	github.com/golang-jwt/jwt/v4 v4.4.3
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/google/uuid v1.3.0
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/klauspost/compress v1.15.14
	github.com/mattn/go-isatty v0.0.16
	github.com/mutagen-io/extstat v0.0.0-20210224131814-32fa3f057fa8
	github.com/mutagen-io/fsevents v0.0.0-20180903111129-10556809b434
	github.com/mutagen-io/gopass v0.0.0-20230214181532-d4b7cdfe054c
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/zeebo/xxh3 v1.0.2
	golang.org/x/net v0.7.0
	golang.org/x/sys v0.5.0
	golang.org/x/text v0.7.0
	google.golang.org/grpc v1.52.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.3
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	golang.org/x/term v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20230119192704-9d59e20e5cd1 // indirect
)

replace k8s.io/apimachinery v0.21.3 => github.com/mutagen-io/apimachinery v0.21.3-mutagen1
