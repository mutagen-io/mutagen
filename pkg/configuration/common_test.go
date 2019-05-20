package configuration

const (
	testConfigurationGibberish = "[a+1a4"
	testConfigurationValid     = `[sync]
mode = "two-way-resolved"
maxEntryCount = 500
maxStagingFileSize = "1000 GB"
probeMode = "assume"
scanMode = "accelerated"
stageMode = "neighboring"

[symlink]
mode = "portable"

[watch]
mode = "force-poll"
pollingInterval = 5

[ignore]
default = ["ignore/this/**", "!ignore/this/that"]

[permissions]
defaultFileMode = 644
defaultDirectoryMode = 0755
defaultOwner = "george"
defaultGroup = "presidents"
`
)
