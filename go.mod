module github.com/mutagen-io/mutagen

go 1.17

require (
	github.com/Microsoft/go-winio v0.5.0
	github.com/bmatcuk/doublestar/v4 v4.0.2
	github.com/compose-spec/compose-go v0.0.0-20210322090015-6166d06f9ce2
	github.com/dustin/go-humanize v1.0.0
	github.com/eknkc/basex v1.0.1
	github.com/fatih/color v1.12.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/joho/godotenv v1.3.0
	github.com/mattn/go-isatty v0.0.13
	github.com/mutagen-io/extstat v0.0.0-20210224131814-32fa3f057fa8
	github.com/mutagen-io/fsevents v0.0.0-20180903111129-10556809b434
	github.com/mutagen-io/gopass v0.0.0-20170602182606-9a121bec1ae7
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/sys v0.0.0-20210819135213-f52c844e1c1c
	golang.org/x/text v0.3.6
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/apimachinery v0.21.3
)

require (
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	google.golang.org/genproto v0.0.0-20210820002220-43fce44e7af1 // indirect
)

replace k8s.io/apimachinery v0.21.3 => github.com/mutagen-io/apimachinery v0.21.3-mutagen1
