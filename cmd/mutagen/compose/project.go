package compose

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mutagen-io/mutagen/pkg/compose"
	"github.com/mutagen-io/mutagen/pkg/forwarding"
	"github.com/mutagen-io/mutagen/pkg/selection"
	forwardingsvc "github.com/mutagen-io/mutagen/pkg/service/forwarding"
	synchronizationsvc "github.com/mutagen-io/mutagen/pkg/service/synchronization"
	"github.com/mutagen-io/mutagen/pkg/synchronization"
	"github.com/mutagen-io/mutagen/pkg/url"
	forwardingurl "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

const (
	// mutagenServiceName is the name used for the Mutagen service.
	mutagenServiceName = "mutagen"
)

// mutagenDockerfile is the Dockerfile template for the Mutagen service.
const mutagenDockerfile = `FROM alpine:latest
RUN ["mkdir", "/volumes"]
ENTRYPOINT ["tail", "-f", "/dev/null"]
`

// mutagenComposeYAML is the Docker Compose configuration template for the
// Mutagen service and any reverse forwarding services.
const mutagenComposeYAML = `version: "{{ .Version }}"
services:
  mutagen:
    build: "{{ .TemporaryDirectory }}/services/mutagen"
    init: true
    # TODO: Add network dependencies
    networks:
    # TODO: Add volume dependencies
    volumes:
  # TODO: Add reverse forwarding services
`

// normalizeProjectNameReplacer is a regular expression used by
// normalizeProjectName to remove unsuitable characters.
var normalizeProjectNameReplacer = regexp.MustCompile(`[^-_a-z0-9]`)

// normalizeProjectName normalizes a project name. It roughly models the logic
// of the normalize_name function inside the get_project_name function in Docker
// Compose.
func normalizeProjectName(name string) string {
	return normalizeProjectNameReplacer.ReplaceAllString(strings.ToLower(name), "")
}

// singleContainerName returns the actual container name for a single-container
// service. It roughly models the logic of the build_container_name function in
// Docker Compose, though it only supports a subset of that functionality.
func singleContainerName(projectName, serviceName string) string {
	return fmt.Sprintf("%s_%s_1", strings.TrimLeft(projectName, "-_"), serviceName)
}

// networkURLPrefix is the lowercase version of the network URL prefix.
const networkURLPrefix = "network://"

// isNetworkURL checks if raw URL is a Docker Compose network pseudo-URL.
func isNetworkURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), networkURLPrefix)
}

// parseNetworkURL parses a Docker Compose network pseudo-URL, converting it to
// a concrete Mutagen Docker URL. It uses the top-level daemon connection flags
// to determine URL parameters and looks for Docker environment variables in the
// fully resolved project environment (which may included variables loaded from
// "dotenv" files). This function also returns the network dependency for the
// URL. This function must only be called on URLs that have been classified as
// network URLs by isNetworkURL, otherwise this function may panic.
func parseNetworkURL(raw string, environment map[string]string, mutagenContainerName string) (*url.URL, string, error) {
	// Strip off the prefix
	raw = raw[len(networkURLPrefix):]

	// Find the first colon, which will indicate the end of the network name.
	var network, endpoint string
	if colonIndex := strings.IndexByte(raw, ':'); colonIndex < 0 {
		return nil, "", errors.New("unable to find forwarding endpoint specification")
	} else if colonIndex == 0 {
		return nil, "", errors.New("empty network name")
	} else {
		network = raw[:colonIndex]
		endpoint = raw[colonIndex+1:]
	}

	// Parse the forwarding endpoint URL to ensure that it's valid.
	if _, _, err := forwardingurl.Parse(endpoint); err != nil {
		return nil, "", fmt.Errorf("invalid forwarding endpoint URL: %w", err)
	}

	// Store any Docker environment variables that we need to preserve. We only
	// store variables that are actually present, because Docker behavior will
	// vary depending on whether a variable is unset vs. set but empty. Note
	// that unlike standard Docker URL parsing, we load these variables from the
	// project environment (which may include values from "dotenv" files). We
	// also don't support endpoint-specific variants since those don't make
	// sense in the context of Docker Compose.
	urlEnvironment := make(map[string]string)
	for _, variable := range url.DockerEnvironmentVariables {
		if value, present := environment[variable]; present {
			urlEnvironment[variable] = value
		}
	}

	// Create a Docker forwarding URL.
	return &url.URL{
		Kind:        url.Kind_Forwarding,
		Protocol:    url.Protocol_Docker,
		Host:        mutagenContainerName,
		Path:        endpoint,
		Environment: urlEnvironment,
		Parameters:  rootConfiguration.DaemonConnectionFlags.ToURLParameters(),
	}, network, nil
}

// mountPathForVolumeInMutagenContainer returns the mount path that will be used
// for a volume inside the Mutagen container. The path will be returned without
// a trailing slash.
// TODO: For Windows support, we'll need to use `c:\volumes\` (or similar).
func mountPathForVolumeInMutagenContainer(volume string) string {
	return "/volumes/" + volume
}

// volumeURLPrefix is the lowercase version of the volume URL prefix.
const volumeURLPrefix = "volume://"

// isVolumeURL checks if raw URL is a Docker Compose volume pseudo-URL.
func isVolumeURL(raw string) bool {
	return strings.HasPrefix(strings.ToLower(raw), volumeURLPrefix)
}

// parseVolumeURL parses a Docker Compose volume pseudo-URL, converting it to a
// concrete Mutagen Docker URL. It uses the top-level daemon connection flags to
// determine URL parameters and looks for Docker environment variables in the
// fully resolved project environment (which may included variables loaded from
// "dotenv" files). This function also returns the volume dependency for the
// URL. This function must only be called on URLs that have been classified as
// volume URLs by isVolumeURL, otherwise this function may panic.
func parseVolumeURL(raw string, environment map[string]string, mutagenContainerName string) (*url.URL, string, error) {
	// Strip off the prefix
	raw = raw[len(volumeURLPrefix):]

	// Find the first slash, which will indicate the end of the volume name. If
	// no slash is found, then we assume that the volume itself is the target
	// synchronization root.
	var volume, path string
	if slashIndex := strings.IndexByte(raw, '/'); slashIndex < 0 {
		volume = raw
		path = mountPathForVolumeInMutagenContainer(volume)
	} else if slashIndex == 0 {
		return nil, "", errors.New("empty volume name")
	} else {
		volume = raw[:slashIndex]
		path = mountPathForVolumeInMutagenContainer(volume) + raw[slashIndex:]
	}

	// Store any Docker environment variables that we need to preserve. We only
	// store variables that are actually present, because Docker behavior will
	// vary depending on whether a variable is unset vs. set but empty. Note
	// that unlike standard Docker URL parsing, we load these variables from the
	// project environment (which may include values from "dotenv" files). We
	// also don't support endpoint-specific variants since those don't make
	// sense in the context of Docker Compose.
	urlEnvironment := make(map[string]string)
	for _, variable := range url.DockerEnvironmentVariables {
		if value, present := environment[variable]; present {
			urlEnvironment[variable] = value
		}
	}

	// Create a Docker synchronization URL.
	return &url.URL{
		Kind:        url.Kind_Synchronization,
		Protocol:    url.Protocol_Docker,
		Host:        mutagenContainerName,
		Path:        path,
		Environment: urlEnvironment,
		Parameters:  rootConfiguration.DaemonConnectionFlags.ToURLParameters(),
	}, volume, nil
}

// project encodes metadata for a Mutagen-enhanced Docker Compose project.
type project struct {
	// environmentFile is the fully resolved absolute path to the environment
	// file that would normally be loaded by Docker Compose. This path is not
	// guaranteed to exist. This value should be passed to Docker Compose
	// commands using the top-level --env-file flag.
	environmentFile string
	// environment is the fully resolved set of project environment variables.
	environment map[string]string
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
	// forwarding are the forwarding session specifications.
	forwarding map[string]*forwardingsvc.CreationSpecification
	// synchronization are the synchronization session specifications.
	synchronization map[string]*synchronizationsvc.CreationSpecification
	// temporaryDirectory is the temorary directory in which generated files are
	// stored for the project.
	temporaryDirectory string
}

// loadProject computes project metadata by emulating Docker Compose's (somewhat
// complex) project resolution behavior. It also generates temporary files
// containing Mutagen image and service definitions. Loading this metadata isn't
// free (though it's not terribly expensive), so it's not done for commands that
// are simple passthroughs. The logic of this loading is a simplified (but
// faithful) emulation of Docker Compose's loading implementation, roughly
// modeling the logic of the project_from_options function. Callers should
// invoke dispose on the resulting project if loading is successful.
func loadProject() (*project, error) {
	// Create a temporary directory to store generated project resources.
	temporaryDirectory, err := ioutil.TempDir("", "io.mutagen.compose.*")
	if err != nil {
		return nil, fmt.Errorf("unable to create temporary directory for project resources: %w", err)
	}

	// Defer removal of the temporary directory in the event that this function
	// is unsuccessful.
	var successful bool
	defer func() {
		if !successful {
			os.RemoveAll(temporaryDirectory)
		}
	}()

	// Compute the fully resolved path to the environment file. If an absolute
	// path has been specified, then it should be used directly. If a relative
	// path has been specified, then it should be treated as relative to the
	// path specified by the --project-directory flag or the current working
	// directory if the --project-directory flag is unspecified. One detail
	// worth noting is that Docker Compose uses os.path.join to compute the
	// final environment path, which will drop any path components prior to an
	// absolute path, unlike Go's path/filepath.Join. For that reason, a manual
	// check for absoluteness is required. This code roughly models the logic of
	// the get_config_from_options and Environment.from_env_file functions in
	// Docker Compose.
	environmentFile := rootConfiguration.envFile
	if environmentFile == "" {
		environmentFile = ".env"
	}
	if filepath.IsAbs(environmentFile) {
		environmentFile = filepath.Clean(environmentFile)
	} else {
		if rootConfiguration.projectDirectory != "" {
			environmentFile = filepath.Join(rootConfiguration.projectDirectory, environmentFile)
		}
		environmentFile, err = filepath.Abs(environmentFile)
		if err != nil {
			return nil, fmt.Errorf("unable to convert environment file path to absolute path: %w", err)
		}
	}

	// Load/compute the effective environment.
	environment, err := compose.LoadEnvironment(environmentFile)
	if err != nil {
		return nil, fmt.Errorf("unable to load/compute environment: %w", err)
	}

	// Check if a project directory has been specified. If so, then convert it
	// to an absolute path. If no project directory was specifed, then it will
	// be computed later once configuration file paths are known.
	projectDirectory := rootConfiguration.projectDirectory
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
	if len(rootConfiguration.file) > 0 {
		files = rootConfiguration.file
	} else if composeFile := environment["COMPOSE_FILE"]; composeFile != "" {
		separator, ok := environment["COMPOSE_PATH_SEPARATOR"]
		if !ok {
			separator = string(os.PathListSeparator)
		} else if separator == "" {
			return nil, errors.New("empty separator specified by COMPOSE_PATH_SEPARATOR")
		}
		files = strings.Split(composeFile, separator)
	}

	// Using the configuration file specifications, determine the final
	// configuration file paths and the project directory (if it wasn't
	// explicitly specified). The three scenarios we need to handle are
	// configuration read from standard input, explicitly specified
	// configuration files, and default configuration file searching behavior.
	// This code roughly models the logic of the config.find function in Docker
	// Compose.
	if len(files) == 1 && files[0] == "-" {
		// Store the standard input stream to a temporary file.
		configurationFilePath := filepath.Join(temporaryDirectory, "standard-input.yaml")
		configurationFile, err := os.Create(configurationFilePath)
		if err != nil {
			return nil, fmt.Errorf("unable to create file to store standard input configuration: %w", err)
		}
		_, err = io.Copy(configurationFile, os.Stdin)
		configurationFile.Close()
		if err != nil {
			return nil, fmt.Errorf("unable to copy standard input configuration: %w", err)
		}
		files = []string{configurationFilePath}

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
		path, name, err := compose.FindDefaultConfigurationFileInPathOrParent(".")
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errors.New("unable to identify configuration file in current directory or any parent")
			}
			return nil, fmt.Errorf("unable to search for Docker Compose configuration file: %w", err)
		}
		files = append(files, filepath.Join(path, name))

		// Search for an override file in the same directory as the primary
		// configuration file.
		if overrideName, err := compose.FindDefaultConfigurationOverrideFileInPath(path); err != nil {
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
	if rootConfiguration.projectName != "" {
		projectName = normalizeProjectName(rootConfiguration.projectName)
	} else if composeProjectName := environment["COMPOSE_PROJECT_NAME"]; composeProjectName != "" {
		projectName = normalizeProjectName(composeProjectName)
	} else if baseName := filepath.Base(projectDirectory); baseName != "" {
		projectName = normalizeProjectName(baseName)
	} else {
		projectName = "default"
	}

	// Load each configuration file, store the version specification for the
	// first file, store Mutagen session configurations, and record service,
	// volume, and network names.
	var version string
	services := make(map[string]struct{})
	volumes := make(map[string]struct{})
	networks := map[string]struct{}{"default": struct{}{}}
	forwardingConfiguration := make(map[string]compose.ForwardingConfiguration)
	synchronizationConfiguration := make(map[string]compose.SynchronizationConfiguration)
	for f, file := range files {
		// Load the configuration file.
		configuration, err := compose.LoadConfiguration(file, environment)
		if err != nil {
			return nil, fmt.Errorf("unable to open configuration file (%s): %w", file, err)
		}

		// Store the version if this is the first configuration file.
		if f == 0 {
			version = configuration.Version
		}

		// Store services, volumes, and networks.
		for name, service := range configuration.Services {
			services[name] = service
		}
		for name, volume := range configuration.Volumes {
			volumes[name] = volume
		}
		for name, network := range configuration.Networks {
			networks[name] = network
		}

		// Store session configurations. We follow standard Docker Compose
		// practice here by letting later session definitions override earlier
		// session definitions with the same names.
		for name, configuration := range configuration.Mutagen.Forwarding {
			forwardingConfiguration[name] = configuration
		}
		for name, configuration := range configuration.Mutagen.Synchronization {
			synchronizationConfiguration[name] = configuration
		}
	}

	// Watch for service name conflicts.
	if _, ok := services[mutagenServiceName]; ok {
		return nil, fmt.Errorf("service name \"%s\" is reserved for Mutagen", mutagenServiceName)
	}

	// Compute the name of the Mutagen service container.
	mutagenContainerName := singleContainerName(projectName, mutagenServiceName)

	// Extract default forwarding session parameters.
	defaultConfigurationForwarding := &forwarding.Configuration{}
	defaultConfigurationSource := &forwarding.Configuration{}
	defaultConfigurationDestination := &forwarding.Configuration{}
	if defaults, ok := forwardingConfiguration["defaults"]; ok {
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
		delete(forwardingConfiguration, "defaults")
	}

	// Extract and validate synchronization defaults.
	defaultConfigurationSynchronization := &synchronization.Configuration{}
	defaultConfigurationAlpha := &synchronization.Configuration{}
	defaultConfigurationBeta := &synchronization.Configuration{}
	if defaults, ok := synchronizationConfiguration["defaults"]; ok {
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
		delete(synchronizationConfiguration, "defaults")
	}

	// Validate forwarding sessions and convert "network://" URLs to their
	// Docker URL equivalents. We'll also take this opportunity to extract the
	// network dependencies for the Mutagen service.
	forwardingSpecifications := make(map[string]*forwardingsvc.CreationSpecification)
	networkDependencies := make(map[string]bool)
	for name, session := range forwardingConfiguration {
		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return nil, fmt.Errorf("invalid forwarding session name (%s): %w", name, err)
		}

		// Parse and validate the source URL. At the moment, we only allow local
		// URLs as forwarding sources since this is the primary use case with
		// Docker Compose. We could support other protocols here, but their
		// usage (especially raw Docker URLs referencing the containers created
		// in this project) is likely to be confusing and error-prone. We may
		// eventually allow network URLs to be sources, but this will require
		// the injection of additional pseudo-services.
		if isNetworkURL(session.Source) {
			return nil, errors.New("network URLs not allowed as forwarding sources")
		}
		sourceURL, err := url.Parse(session.Source, url.Kind_Forwarding, true)
		if err != nil {
			return nil, fmt.Errorf("unable to parse forwarding source URL (%s): %w", session.Source, err)
		} else if sourceURL.Protocol != url.Protocol_Local {
			return nil, errors.New("only local URLs allowed as forwarding sources")
		}

		// Parse and validate the destination URL. At the moment, we only allow
		// network URLs as forwarding destinations since this is the primary use
		// case with Docker Compose. We could support other protocols here, but
		// we don't at the moment for the reasons outlined above.
		if !isNetworkURL(session.Destination) {
			return nil, errors.New("forwarding destinations should be network URLs")
		}
		destinationURL, network, err := parseNetworkURL(session.Destination, environment, mutagenContainerName)
		if err != nil {
			return nil, fmt.Errorf("unable to parse forwarding destination URL (%s): %w", session.Destination, err)
		}
		networkDependencies[network] = true

		// Compute configuration.
		configuration := session.Configuration.Configuration()
		if err := configuration.EnsureValid(false); err != nil {
			return nil, fmt.Errorf("invalid forwarding session configuration for %s: %w", name, err)
		}
		configuration = forwarding.MergeConfigurations(defaultConfigurationForwarding, configuration)

		// Compute source-specific configuration.
		sourceConfiguration := session.ConfigurationSource.Configuration()
		if err := sourceConfiguration.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid forwarding session source configuration for %s: %w", name, err)
		}
		sourceConfiguration = forwarding.MergeConfigurations(defaultConfigurationSource, sourceConfiguration)

		// Compute destination-specific configuration.
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
			Labels:                   map[string]string{
				// TODO: Compute and set labels.
			},
		}
	}

	// Validate synchronization sessions and convert "volume://" URLs to their
	// Docker URL equivalents. We'll also take this opportunity to extract the
	// volume dependencies for the Mutagen service.
	synchronizationSpecifications := make(map[string]*synchronizationsvc.CreationSpecification)
	volumeDependencies := make(map[string]bool)
	for name, session := range synchronizationConfiguration {
		// Verify that the name is valid.
		if err := selection.EnsureNameValid(name); err != nil {
			return nil, fmt.Errorf("invalid synchronization session name (%s): %v", name, err)
		}

		// Enforce that exactly one of the alpha and beta URLs is a volume URL.
		// At the moment, we only support synchronization sessions where one of
		// the URLs is local and one of the URLs is a volume URL. We could
		// support other combinations here, but their usage (especialy raw
		// Docker URLs referencing the containers created in this project) is
		// likely to be confusing and error-prone. We may change this in the
		// future.
		alphaIsVolume := isVolumeURL(session.Alpha)
		betaIsVolume := isVolumeURL(session.Beta)
		if !(alphaIsVolume || betaIsVolume) {
			return nil, fmt.Errorf("neither alpha nor beta references a volume in synchronization session (%s)", name)
		} else if alphaIsVolume && betaIsVolume {
			return nil, fmt.Errorf("both alpha and beta reference volumes in synchronization session (%s)", name)
		}

		// Parse and validate the alpha URL. If it isn't a volume URL, then it
		// must be a local URL.
		var alphaURL *url.URL
		if alphaIsVolume {
			if a, volume, err := parseVolumeURL(session.Alpha, environment, mutagenContainerName); err != nil {
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
		}

		// Parse and validate the beta URL. If it isn't a volume URL, then it
		// must be a local URL.
		var betaURL *url.URL
		if betaIsVolume {
			if b, volume, err := parseVolumeURL(session.Beta, environment, mutagenContainerName); err != nil {
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
		}

		// Compute configuration.
		configuration := session.Configuration.Configuration()
		if err := configuration.EnsureValid(false); err != nil {
			return nil, fmt.Errorf("invalid synchronization session configuration for %s: %v", name, err)
		}
		configuration = synchronization.MergeConfigurations(defaultConfigurationSynchronization, configuration)

		// Compute alpha-specific configuration.
		alphaConfiguration := session.ConfigurationAlpha.Configuration()
		if err := alphaConfiguration.EnsureValid(true); err != nil {
			return nil, fmt.Errorf("invalid synchronization session alpha configuration for %s: %v", name, err)
		}
		alphaConfiguration = synchronization.MergeConfigurations(defaultConfigurationAlpha, alphaConfiguration)

		// Compute beta-specific configuration.
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
			Labels:             map[string]string{
				// TODO: Compute and set labels.
			},
		}
	}

	// Generate the Mutagen Dockerfile and Docker Compose configuration file.
	// TODO: Implement.
	_ = version
	_ = networkDependencies
	_ = volumeDependencies

	// Success.
	successful = true
	return &project{
		environmentFile:    environmentFile,
		environment:        environment,
		files:              files,
		workingDirectory:   projectDirectory,
		name:               projectName,
		forwarding:         forwardingSpecifications,
		synchronization:    synchronizationSpecifications,
		temporaryDirectory: temporaryDirectory,
	}, nil
}

// dispose removes any temporary generated project files from disk.
func (p *project) dispose() error {
	return os.RemoveAll(p.temporaryDirectory)
}

// topLevelFlags returns a slice of top-level project flags (namely -f/--file,
// -p/--project-name, --project-directory, and --env-file) that should be used
// in conjunction with those from the global topLevelFlags result (with the
// global topLevelFlags function being called with excludeProjectFlags set to
// true).
func (p *project) topLevelFlags() []string {
	// Preallocate flag storage.
	flags := make([]string, 0, 2*len(p.files)+2+2+2)

	// Add flags.
	for _, file := range p.files {
		flags = append(flags, "--file", file)
	}
	flags = append(flags, "--project-name", p.name)
	flags = append(flags, "--project-directory", p.workingDirectory)
	flags = append(flags, "--env-file", p.environmentFile)

	// Done.
	return flags
}
