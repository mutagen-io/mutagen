package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/compose/internal/configuration"
	"github.com/mutagen-io/mutagen/pkg/docker"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/mutagen"
	"github.com/mutagen-io/mutagen/pkg/selection"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/sidecar"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/url"
	forwardingurl "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

const (
	// MutagenServiceName is the name used for the Mutagen service in
	// Mutagen-enhanced Docker Compose Projects.
	MutagenServiceName = "mutagen"
	// MutagenProfileName is the name use for the Mutagen profile in
	// Mutagen-enhanced Docker Compose Projects.
	MutagenProfileName = "mutagen_internal"
)

// mutagenSidecarImage is the Mutagen sidecar image tag.
var mutagenSidecarImage string

func init() {
	// Compute the sidecar image tag.
	mutagenSidecarImage = sidecar.BaseTag + ":" + mutagen.Version
}

// normalizeProjectNameReplacer is a regular expression used by
// normalizeProjectName to remove unsuitable characters.
var normalizeProjectNameReplacer = regexp.MustCompile(`[^-_a-z0-9]`)

// normalizeProjectName normalizes a project name. It roughly models the logic
// of the normalize_name function inside the get_project_name function in Docker
// Compose.
func normalizeProjectName(name string) string {
	return normalizeProjectNameReplacer.ReplaceAllString(strings.ToLower(name), "")
}

// daemonIdentifierNormalizers are pairs of matchers and normalization functions
// that can be used to convert daemon identifiers to a format suitable for use
// with Mutagen session labels. Since the Docker daemon format isn't guaranteed
// to be stable, we use regular expressions to ensure that we understand the
// format. All normalizers in this list should be cognizant of label value
// character and length restrictions.
var daemonIdentifierNormalizers = []struct {
	matcher    *regexp.Regexp
	normalizer func(string) string
}{
	{
		regexp.MustCompile(`^([A-Z0-9]{4}:){11}[A-Z0-9]{4}$`),
		func(s string) string { return strings.ReplaceAll(s, ":", "_") },
	},
}

// singleContainerName returns the actual container name for a single-container
// service. It roughly models the logic of the build_container_name function in
// Docker Compose, though it only supports a subset of that functionality.
func singleContainerName(projectName, serviceName string) string {
	return fmt.Sprintf("%s_%s_1", strings.TrimLeft(projectName, "-_"), serviceName)
}

// Project encodes metadata for a Mutagen-enhanced Docker Compose project.
type Project struct {
	// environmentFile is the fully resolved absolute path to the environment
	// file that would normally be loaded by Docker Compose. This path is not
	// guaranteed to exist. This value should be passed to Docker Compose
	// commands using the top-level --env-file flag.
	environmentFile string
	// files are the fully resolved absolute paths to the configuration files
	// for the project. The base set of files is determined using the -f/--file
	// command line flag(s), the COMPOSE_FILE and COMPOSE_PATH_SEPARATOR
	// environment variables, or default filesystem searching, in that order of
	// preference. Each specified path is converted to an absolute path based on
	// Docker Compose's resolution behavior. If these specifications indicate
	// that configuration should be read from standard input, then the contents
	// of standard input will be stored on disk in a temporary file and that
	// file will take the place of the provided specification. The last element
	// of this slice will be the generated Mutagen service configuration file.
	// These values should be passed to Docker Compose commands using top-level
	// -f/--file flag(s).
	files []string
	// workingDirectory is the fully resolved project working directory. This
	// value should be passed to Docker Compose commands using the top-level
	// --project-directory flag.
	workingDirectory string
	// name is the fully resolved project name. This value should be passed to
	// Docker Compose commands using the top-level --name flag.
	name string
	// Forwarding are the forwarding session specifications.
	Forwarding map[string]*forwardingsvc.CreationSpecification
	// Synchronization are the synchronization session specifications.
	Synchronization map[string]*synchronizationsvc.CreationSpecification
	// daemonIdentifier is the normalized Docker daemon identifier.
	daemonIdentifier string
	// temporaryDirectory is the temorary directory in which generated files are
	// stored for the project.
	temporaryDirectory string
}

// LoadProject computes Docker Compose project metadata, loads the project's
// configuration files, extracts Mutagen session definitions from "x-mutagen"
// extensions, and generates temporary files containing Mutagen image and
// service definitions. The logic of this loading is a simplified (but faithful)
// emulation of Docker Compose's loading implementation, roughly modeling the
// logic of the project_from_options function. Callers should invoke Dispose on
// the resulting project if loading is successful.
func LoadProject(projectFlags ProjectFlags, daemonFlags docker.DaemonConnectionFlags) (*Project, error) {
	// Create a temporary directory to store generated project resources.
	temporaryDirectory, err := os.MkdirTemp("", "io.mutagen.compose.*")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary directory for project resources: %w", err)
	}

	// Defer removal of the temporary directory in the event that initialization
	// is unsuccessful.
	var successful bool
	defer func() {
		if !successful {
			os.RemoveAll(temporaryDirectory)
		}
	}()

	// Compute the fully resolved path to the environment file and load/compute
	// the effective environment. If an absolute path has been specified for the
	// environment file, then it should be used directly. If a relative path has
	// been specified, then it should be treated as relative to the path
	// specified by the --project-directory flag or the current working
	// directory if the --project-directory flag is unspecified. One detail
	// worth noting is that Docker Compose uses os.path.join to compute the
	// final environment path, which will drop any path components prior to an
	// absolute path, unlike Go's path/filepath.Join. For that reason, a manual
	// check for absoluteness is required. This code roughly models the logic of
	// the get_config_from_options and Environment.from_env_file functions in
	// Docker Compose.
	environmentFile := projectFlags.EnvFile
	if environmentFile == "" {
		environmentFile = ".env"
	}
	if filepath.IsAbs(environmentFile) {
		environmentFile = filepath.Clean(environmentFile)
	} else {
		if projectFlags.ProjectDirectory != "" {
			environmentFile = filepath.Join(projectFlags.ProjectDirectory, environmentFile)
		}
		environmentFile, err = filepath.Abs(environmentFile)
		if err != nil {
			return nil, fmt.Errorf("unable to convert environment file path to absolute path: %w", err)
		}
	}
	environment, err := loadEnvironment(environmentFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load/compute environment: %w", err)
	}
	if _, err := os.Lstat(environmentFile); err != nil {
		if os.IsNotExist(err) {
			environmentFile = ""
		} else {
			return nil, fmt.Errorf("unable to check environment file existence: %w", err)
		}
	}

	// Query the Docker daemon metadata and ensure that the Docker daemon is
	// running an OS supported by Mutagen's Docker Compose integration.
	daemonMetadata, err := docker.GetDaemonMetadata(daemonFlags, environment)
	if err != nil {
		return nil, fmt.Errorf("unable to query Docker daemon metadata: %w", err)
	} else if !isSupportedPlatform(daemonMetadata.Platform) {
		return nil, fmt.Errorf("unsupported Docker platform: %s", daemonMetadata.Platform)
	}

	// Create a daemon identifier suitable for use with session labels.
	var daemonIdentifier string
	for _, pair := range daemonIdentifierNormalizers {
		if pair.matcher.MatchString(daemonMetadata.Identifier) {
			daemonIdentifier = pair.normalizer(daemonMetadata.Identifier)
		}
	}
	if daemonIdentifier == "" {
		return nil, fmt.Errorf("unknown daemon identifier format: %s", daemonMetadata.Identifier)
	}

	// Check if a project directory has been specified. If so, then convert it
	// to an absolute path. If no project directory was specified, then it will
	// be computed later once configuration file paths are known.
	projectDirectory := projectFlags.ProjectDirectory
	if projectDirectory != "" {
		if projectDirectory, err = filepath.Abs(projectDirectory); err != nil {
			return nil, fmt.Errorf("unable to convert project directory (%s) to absolute path: %w", projectDirectory, err)
		}
	}

	// Identify any explicit configuration file specifications. This isn't the
	// same as determining the final configuration file paths, we're just
	// determining where we should look for explicit specifications (i.e. on the
	// command line or in the environment) and the value of those specifications
	// if provided. There may not be any explicit specifications (indicating
	// that default search behavior should be used). This code roughly models
	// the logic of the get_config_path_from_options function in Docker Compose.
	var files []string
	if len(projectFlags.File) > 0 {
		files = projectFlags.File
	} else if composeFile := environment["COMPOSE_FILE"]; composeFile != "" {
		separator, ok := environment["COMPOSE_PATH_SEPARATOR"]
		if !ok {
			separator = string(os.PathListSeparator)
		} else if separator == "" {
			return nil, errors.New("empty separator specified by COMPOSE_PATH_SEPARATOR")
		}
		files = strings.Split(composeFile, separator)
	}

	// Using the configuration file specifications, determine the configuration
	// file paths and the project directory (if it wasn't explicitly specified).
	// The three scenarios we need to handle are configuration read from
	// standard input, explicitly specified configuration files, and default
	// configuration file searching behavior. This code roughly models the logic
	// of the config.find function in Docker Compose.
	if len(files) == 1 && files[0] == "-" {
		// Store the standard input stream to a temporary file.
		standardInputPath := filepath.Join(temporaryDirectory, "standard-input.yaml")
		if err := storeStandardInput(standardInputPath); err != nil {
			return nil, fmt.Errorf("unable to store configuration from standard input: %w", err)
		}
		files = []string{standardInputPath}

		// If a project directory wasn't explicitly specified, then use the
		// current working directory.
		if projectDirectory == "" {
			if projectDirectory, err = os.Getwd(); err != nil {
				return nil, fmt.Errorf("unable to determine current working directory: %w", err)
			}
		}
	} else if len(files) > 0 {
		// Convert files to absolute paths. Explicit file specifications are
		// always treated as relative to the current working directory, even if
		// a project working directory has been explicitly specified.
		for f, file := range files {
			if files[f], err = filepath.Abs(file); err != nil {
				return nil, fmt.Errorf("unable to convert file specification (%s) to absolute path: %w", file, err)
			}
		}

		// If a project directory wasn't explicitly specified, then use the
		// parent directory of the first configuration file.
		if projectDirectory == "" {
			projectDirectory = filepath.Dir(files[0])
		}
	} else {
		// Search for a configuration file in the current directory and its
		// parent directories.
		path, name, err := findDefaultConfigurationFileInPathOrParent(".")
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("unable to identify configuration file in current directory or any parent")
			}
			return nil, fmt.Errorf("unable to search for Docker Compose configuration file: %w", err)
		}
		files = append(files, filepath.Join(path, name))

		// Search for an override file in the same directory as the primary
		// configuration file.
		if overrideName, err := findDefaultConfigurationOverrideFileInPath(path); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("unable to identify configuration override file: %w", err)
			}
		} else {
			files = append(files, filepath.Join(path, overrideName))
		}

		// If a project directory wasn't explicitly specified, then use the path
		// of the configuration file.
		if projectDirectory == "" {
			projectDirectory = path
		}
	}

	// Determine the project name. This code roughly models the logic of the
	// get_project_name function in Docker Compose.
	var projectName string
	if projectFlags.ProjectName != "" {
		projectName = normalizeProjectName(projectFlags.ProjectName)
	} else if composeProjectName := environment["COMPOSE_PROJECT_NAME"]; composeProjectName != "" {
		projectName = normalizeProjectName(composeProjectName)
	} else if baseName := filepath.Base(projectDirectory); baseName != "" {
		projectName = normalizeProjectName(baseName)
	} else {
		projectName = "default"
	}

	// Load each configuration file, recording service, volume, and network
	// names, and storing Mutagen session configurations.
	services := make(map[string]bool)
	volumes := make(map[string]bool)
	networks := map[string]bool{"default": true}
	sessions := configuration.MutagenConfiguration{
		Forwarding:      make(map[string]configuration.ForwardingConfiguration),
		Synchronization: make(map[string]configuration.SynchronizationConfiguration),
	}
	for _, file := range files {
		// Load the configuration file.
		configuration, err := configuration.Load(file, environment)
		if err != nil {
			return nil, fmt.Errorf("unable to load configuration file (%s): %w", file, err)
		}

		// Record the service, volume, and network names defined in the file.
		for name := range configuration.Services {
			services[name] = true
		}
		for name := range configuration.Volumes {
			volumes[name] = true
		}
		for name := range configuration.Networks {
			networks[name] = true
		}

		// Store session configurations. We follow standard Docker Compose
		// practice here by letting later session definitions override earlier
		// session definitions with the same names.
		for name, configuration := range configuration.Mutagen.Forwarding {
			sessions.Forwarding[name] = configuration
		}
		for name, configuration := range configuration.Mutagen.Synchronization {
			sessions.Synchronization[name] = configuration
		}
	}

	// Watch for service name conflicts.
	if services[MutagenServiceName] {
		return nil, fmt.Errorf("service name \"%s\" is reserved for Mutagen", MutagenServiceName)
	}

	// Compute the name of the Mutagen service container.
	mutagenContainerName := singleContainerName(projectName, MutagenServiceName)

	// Extract default forwarding session parameters.
	defaultConfigurationForwarding := &forwarding.Configuration{}
	defaultConfigurationSource := &forwarding.Configuration{}
	defaultConfigurationDestination := &forwarding.Configuration{}
	if defaults, ok := sessions.Forwarding["defaults"]; ok {
		if defaults.Source != "" {
			return nil, errors.New("source URL not allowed in default forwarding configuration")
		} else if defaults.Destination != "" {
			return nil, errors.New("destination URL not allowed in default forwarding configuration")
		}
		defaultConfigurationForwarding = defaults.Configuration.Configuration()
		if err := defaultConfigurationForwarding.EnsureValid(false); err != nil {
			return nil, fmt.Errorf("invalid default forwarding configuration: %w", err)
		}
		defaultConfigurationSource = defaults.ConfigurationSource.Configuration()
		if err := defaultConfigurationSource.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid default forwarding source configuration: %w", err)
		}
		defaultConfigurationDestination = defaults.ConfigurationDestination.Configuration()
		if err := defaultConfigurationDestination.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid default forwarding destination configuration: %w", err)
		}
		delete(sessions.Forwarding, "defaults")
	}

	// Extract and validate synchronization defaults.
	defaultConfigurationSynchronization := &synchronization.Configuration{}
	defaultConfigurationAlpha := &synchronization.Configuration{}
	defaultConfigurationBeta := &synchronization.Configuration{}
	if defaults, ok := sessions.Synchronization["defaults"]; ok {
		if defaults.Alpha != "" {
			return nil, errors.New("alpha URL not allowed in default synchronization configuration")
		} else if defaults.Beta != "" {
			return nil, errors.New("beta URL not allowed in default synchronization configuration")
		}
		defaultConfigurationSynchronization = defaults.Configuration.Configuration()
		if err := defaultConfigurationSynchronization.EnsureValid(false); err != nil {
			return nil, fmt.Errorf("invalid default synchronization configuration: %w", err)
		}
		defaultConfigurationAlpha = defaults.ConfigurationAlpha.Configuration()
		if err := defaultConfigurationAlpha.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid default synchronization alpha configuration: %w", err)
		}
		defaultConfigurationBeta = defaults.ConfigurationBeta.Configuration()
		if err := defaultConfigurationBeta.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid default synchronization beta configuration: %w", err)
		}
		delete(sessions.Synchronization, "defaults")
	}

	// Validate forwarding configurations, convert them to session creation
	// specifications, and extract network dependencies for the Mutagen service.
	forwardingSpecifications := make(map[string]*forwardingsvc.CreationSpecification)
	networkDependencies := make(map[string]bool)
	for name, session := range sessions.Forwarding {
		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return nil, fmt.Errorf("invalid forwarding session name (%s): %w", name, err)
		}

		// Parse and validate the source URL. At the moment, we only allow local
		// URLs as forwarding sources since this is the primary use case with
		// Docker Compose. Supporting reverse forwarding is somewhat ill-defined
		// and would require the injection of additional services to intercept
		// traffic (though we may support this in the future). We also avoid
		// other protocols (such as SSH and Docker) since they're likely to be
		// confusing and error-prone (especially raw Docker URLs referencing
		// containers in this project that won't play nicely with container
		// startup ordering). Finally, we only support TCP-based endpoints since
		// they constitute the primary use case with Docker Compose and because
		// other protocols would likely be error-prone and require
		// project-relative path resolution.
		if isNetworkURL(session.Source) {
			return nil, fmt.Errorf("network URL (%s) not allowed as forwarding source", session.Source)
		}
		sourceURL, err := url.Parse(session.Source, url.Kind_Forwarding, true)
		if err != nil {
			return nil, fmt.Errorf("unable to parse forwarding source URL (%s): %w", session.Source, err)
		} else if sourceURL.Protocol != url.Protocol_Local {
			return nil, errors.New("only local URLs allowed as forwarding sources")
		} else if protocol, _, err := forwardingurl.Parse(sourceURL.Path); err != nil {
			panic("forwarding URL failed to reparse")
		} else if !isTCPForwardingProtocol(protocol) {
			return nil, fmt.Errorf("non-TCP-based forwarding endpoint (%s) unsupported", sourceURL.Path)
		}

		// Parse and validate the destination URL. At the moment, we only allow
		// network pseudo-URLs (with TCP-based endpoints) as forwarding
		// destinations for the reasons outlined above. The parseNetworkURL will
		// enforce that a TCP-based forwarding endpoint is used.
		if !isNetworkURL(session.Destination) {
			return nil, fmt.Errorf("forwarding destination (%s) should be a network URL", session.Destination)
		}
		destinationURL, network, err := parseNetworkURL(session.Destination, mutagenContainerName, environment, daemonFlags)
		if err != nil {
			return nil, fmt.Errorf("unable to parse forwarding destination URL (%s): %w", session.Destination, err)
		}
		networkDependencies[network] = true

		// Compute the session configuration.
		configuration := session.Configuration.Configuration()
		if err := configuration.EnsureValid(false); err != nil {
			return nil, fmt.Errorf("invalid forwarding session configuration for %s: %w", name, err)
		}
		configuration = forwarding.MergeConfigurations(defaultConfigurationForwarding, configuration)

		// Compute the source-specific configuration.
		sourceConfiguration := session.ConfigurationSource.Configuration()
		if err := sourceConfiguration.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid forwarding session source configuration for %s: %w", name, err)
		}
		sourceConfiguration = forwarding.MergeConfigurations(defaultConfigurationSource, sourceConfiguration)

		// Compute the destination-specific configuration.
		destinationConfiguration := session.ConfigurationDestination.Configuration()
		if err := destinationConfiguration.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid forwarding session destination configuration for %s: %w", name, err)
		}
		destinationConfiguration = forwarding.MergeConfigurations(defaultConfigurationDestination, destinationConfiguration)

		// Record the specification.
		forwardingSpecifications[name] = &forwardingsvc.CreationSpecification{
			Source:                   sourceURL,
			Destination:              destinationURL,
			Configuration:            configuration,
			ConfigurationSource:      sourceConfiguration,
			ConfigurationDestination: destinationConfiguration,
			Name:                     name,
			Labels: map[string]string{
				projectNameLabel:      projectName,
				daemonIdentifierLabel: daemonIdentifier,
			},
		}
	}

	// Validate synchronization configurations, convert them to session creation
	// specifications, and extract volume dependencies for the Mutagen service.
	synchronizationSpecifications := make(map[string]*synchronizationsvc.CreationSpecification)
	volumeDependencies := make(map[string]bool)
	for name, session := range sessions.Synchronization {
		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return nil, fmt.Errorf("invalid synchronization session name (%s): %v", name, err)
		}

		// Enforce that exactly one of the session URLs is a volume URL. At the
		// moment, we only support synchronization sessions where one of the
		// URLs is local the other is a volume URL. We'll check that the
		// non-volume URL is local when parsing. We could support other protocol
		// combinations for synchronization (and we may in the future), but for
		// now we're focused on supporting the primary Docker Compose use case
		// and avoiding the confusing and error-prone cases described above.
		alphaIsVolume := isVolumeURL(session.Alpha)
		betaIsVolume := isVolumeURL(session.Beta)
		if !(alphaIsVolume || betaIsVolume) {
			return nil, fmt.Errorf("neither alpha nor beta references a volume in synchronization session (%s)", name)
		} else if alphaIsVolume && betaIsVolume {
			return nil, fmt.Errorf("both alpha and beta reference volumes in synchronization session (%s)", name)
		}

		// Parse and validate the alpha URL. If it isn't a volume URL, then it
		// must be a local URL. In the case of a local URL, we treat relative
		// paths as relative to the project directory, so we have to override
		// the default URL parsing behavior in that case.
		var alphaURL *url.URL
		if alphaIsVolume {
			if a, volume, err := parseVolumeURL(session.Alpha, daemonMetadata.Platform, mutagenContainerName, environment, daemonFlags); err != nil {
				return nil, fmt.Errorf("unable to parse synchronization alpha URL (%s): %w", session.Alpha, err)
			} else {
				alphaURL = a
				volumeDependencies[volume] = true
			}
		} else {
			alphaURL, err = url.Parse(session.Alpha, url.Kind_Synchronization, true)
			if err != nil {
				return nil, fmt.Errorf("unable to parse synchronization alpha URL (%s): %w", session.Alpha, err)
			} else if alphaURL.Protocol != url.Protocol_Local {
				return nil, errors.New("only local and volume URLs allowed as synchronization URLs")
			}
			if !filepath.IsAbs(session.Alpha) {
				if alphaURL.Path, err = filepath.Abs(filepath.Join(projectDirectory, session.Alpha)); err != nil {
					return nil, fmt.Errorf("unable to resolve relative alpha URL (%s): %w", session.Alpha, err)
				}
			}
		}

		// Parse and validate the beta URL using the same strategy.
		var betaURL *url.URL
		if betaIsVolume {
			if b, volume, err := parseVolumeURL(session.Beta, daemonMetadata.Platform, mutagenContainerName, environment, daemonFlags); err != nil {
				return nil, fmt.Errorf("unable to parse synchronization beta URL (%s): %w", session.Beta, err)
			} else {
				betaURL = b
				volumeDependencies[volume] = true
			}
		} else {
			betaURL, err = url.Parse(session.Beta, url.Kind_Synchronization, false)
			if err != nil {
				return nil, fmt.Errorf("unable to parse synchronization beta URL (%s): %w", session.Beta, err)
			} else if betaURL.Protocol != url.Protocol_Local {
				return nil, errors.New("only local and volume URLs allowed as synchronization URLs")
			}
			if !filepath.IsAbs(session.Beta) {
				if betaURL.Path, err = filepath.Abs(filepath.Join(projectDirectory, session.Beta)); err != nil {
					return nil, fmt.Errorf("unable to resolve relative beta URL (%s): %w", session.Beta, err)
				}
			}
		}

		// Compute the session configuration.
		configuration := session.Configuration.Configuration()
		if err := configuration.EnsureValid(false); err != nil {
			return nil, fmt.Errorf("invalid synchronization session configuration for %s: %v", name, err)
		}
		configuration = synchronization.MergeConfigurations(defaultConfigurationSynchronization, configuration)

		// Compute the alpha-specific configuration.
		alphaConfiguration := session.ConfigurationAlpha.Configuration()
		if err := alphaConfiguration.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid synchronization session alpha configuration for %s: %v", name, err)
		}
		alphaConfiguration = synchronization.MergeConfigurations(defaultConfigurationAlpha, alphaConfiguration)

		// Compute the beta-specific configuration.
		betaConfiguration := session.ConfigurationBeta.Configuration()
		if err := betaConfiguration.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid synchronization session beta configuration for %s: %v", name, err)
		}
		betaConfiguration = synchronization.MergeConfigurations(defaultConfigurationBeta, betaConfiguration)

		// Record the specification.
		synchronizationSpecifications[name] = &synchronizationsvc.CreationSpecification{
			Alpha:              alphaURL,
			Beta:               betaURL,
			Configuration:      configuration,
			ConfigurationAlpha: alphaConfiguration,
			ConfigurationBeta:  betaConfiguration,
			Name:               name,
			Labels: map[string]string{
				projectNameLabel:      projectName,
				daemonIdentifierLabel: daemonIdentifier,
			},
		}
	}

	// Validate network and volume dependencies.
	for network := range networkDependencies {
		if !networks[network] {
			return nil, fmt.Errorf("undefined network (%s) referenced by forwarding session", network)
		}
	}
	for volume := range volumeDependencies {
		if !volumes[volume] {
			return nil, fmt.Errorf("undefined volume (%s) referenced by synchronization session", volume)
		}
	}

	// Generate the Mutagen service configuration.
	mutagenServiceConfiguration := &configuration.GeneratedServiceConfiguration{
		Image:    mutagenSidecarImage,
		Profiles: []string{MutagenProfileName},
	}
	for network := range networkDependencies {
		mutagenServiceConfiguration.Networks = append(mutagenServiceConfiguration.Networks,
			network,
		)
	}
	for volume := range volumeDependencies {
		mutagenServiceConfiguration.Volumes = append(mutagenServiceConfiguration.Volumes,
			volume+":"+mountPathForVolumeInMutagenContainer(daemonMetadata.Platform, volume),
		)
	}

	// Generate the Mutagen Docker Compose configuration file.
	mutagenComposeConfiguration := &configuration.GeneratedComposeConfiguration{
		Services: map[string]*configuration.GeneratedServiceConfiguration{
			MutagenServiceName: mutagenServiceConfiguration,
		},
	}
	mutagenComposeConfigurationPath := filepath.Join(temporaryDirectory, "mutagen.yml")
	if err := mutagenComposeConfiguration.Store(mutagenComposeConfigurationPath); err != nil {
		return nil, fmt.Errorf("unable to store Docker Compose configuration for Mutagen service(s): %w", err)
	}
	files = append(files, mutagenComposeConfigurationPath)

	// Success.
	successful = true
	return &Project{
		environmentFile:    environmentFile,
		files:              files,
		workingDirectory:   projectDirectory,
		name:               projectName,
		Forwarding:         forwardingSpecifications,
		Synchronization:    synchronizationSpecifications,
		daemonIdentifier:   daemonIdentifier,
		temporaryDirectory: temporaryDirectory,
	}, nil
}

// Dispose removes any temporary generated project files from disk.
func (p *Project) Dispose() error {
	return os.RemoveAll(p.temporaryDirectory)
}

// TopLevelFlags returns a slice of top-level project flags (namely -f/--file,
// -p/--project-name, --project-directory, and --env-file) with fully resolved
// values.
func (p *Project) TopLevelFlags() []string {
	// Preallocate flag storage.
	flags := make([]string, 0, 2*len(p.files)+2+2+2)

	// Add flags.
	for _, file := range p.files {
		flags = append(flags, "--file", file)
	}
	flags = append(flags, "--project-name", p.name)
	flags = append(flags, "--project-directory", p.workingDirectory)
	if p.environmentFile != "" {
		flags = append(flags, "--env-file", p.environmentFile)
	}

	// Done.
	return flags
}

// SessionSelection returns a Mutagen session selection that uses a label
// selector to identify sessions associated with the project.
func (p *Project) SessionSelection() *selection.Selection {
	return &selection.Selection{
		LabelSelector: fmt.Sprintf("%s == %s,%s == %s",
			projectNameLabel, p.name,
			daemonIdentifierLabel, p.daemonIdentifier,
		),
	}
}
