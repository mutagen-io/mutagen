module github.com/mutagen-io/mutagen

go 1.19

require (
	github.com/Microsoft/go-winio v0.5.2
	github.com/bmatcuk/doublestar/v4 v4.2.0
	github.com/dustin/go-humanize v1.0.0
	github.com/eknkc/basex v1.0.1
	github.com/fatih/color v1.13.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/google/uuid v1.3.0
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/mattn/go-isatty v0.0.16
	github.com/mutagen-io/extstat v0.0.0-20210224131814-32fa3f057fa8
	github.com/mutagen-io/fsevents v0.0.0-20180903111129-10556809b434
	github.com/mutagen-io/gopass v0.0.0-20170602182606-9a121bec1ae7
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/net v0.0.0-20221004154528-8021a29435af
	golang.org/x/sys v0.0.0-20220928140112-f11e5e49a4ec
	golang.org/x/text v0.3.7
	google.golang.org/grpc v1.49.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.3
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	golang.org/x/crypto v0.0.0-20221005025214-4161e89ecf1b // indirect
	golang.org/x/term v0.0.0-20220919170432-7a66f970e087 // indirect
	google.golang.org/genproto v0.0.0-20220930163606-c98284e70a91 // indirect
)

replace k8s.io/apimachinery v0.21.3 => github.com/mutagen-io/apimachinery v0.21.3-mutagen1
