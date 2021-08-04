module github.com/mutagen-io/mutagen

go 1.16

require (
	github.com/Microsoft/go-winio v0.4.16
	github.com/bmatcuk/doublestar v1.1.1
	github.com/compose-spec/compose-go v0.0.0-20210322090015-6166d06f9ce2
	github.com/dustin/go-humanize v1.0.0
	github.com/eknkc/basex v1.0.0
	github.com/fatih/color v1.10.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.1
	github.com/google/uuid v1.2.0
	github.com/hashicorp/yamux v0.0.0-20210316155119-a95892c5f864
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/joho/godotenv v1.3.0
	github.com/mattn/go-isatty v0.0.12
	github.com/mutagen-io/extstat v0.0.0-20210224131814-32fa3f057fa8
	github.com/mutagen-io/fsevents v0.0.0-20180903111129-10556809b434
	github.com/mutagen-io/gopass v0.0.0-20170602182606-9a121bec1ae7
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.0.0-20210322153248-0c34fe9e7dc2 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887
	golang.org/x/text v0.3.6
	google.golang.org/grpc v1.36.1
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/apimachinery v0.21.3
)

replace k8s.io/apimachinery v0.21.3 => github.com/mutagen-io/apimachinery v0.21.3-mutagen1
