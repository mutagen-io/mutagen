package compose

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"

	"github.com/spf13/cobra"

	"github.com/mutagen-io/mutagen/cmd"
)

// defaultConfigurationFileNames are the file names used when searching the
// current directory and parent directories for Docker Compose configuration
// files.
var defaultConfigurationFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
}

// defaultConfigurationOverrideFileNames are the names used when searching for
// override configuration files alongside the nominal configuration file.
var defaultConfigurationOverrideFileNames = []string{
	"docker-compose.override.yml",
	"docker-compose.override.yaml",
}

// findConfigurationFileInPathOrParent searches the specified path and its
// parent directories for a default Docker Compose configuration file, stopping
// after the first match. It returns the path at which the match was found and
// the matching file name. It will return os.ErrNotExist if no match is found,
// as well as any other error that occurs while traversing the filesystem. The
// specified path will be converted to an absolute path and cleaned, and thus
// any resulting path will also be absolute and cleaned. This function rougly
// models the logic of the find_candidates_in_parent_dirs function in Docker
// Compose. It's worth noting that find_candidates_in_parent_dirs will allow
// multiple matches (unlike get_default_override_file) and will just use the
// first match (while printing a warning). We do the same in this function,
// except that we don't print a warning.
func findConfigurationFileInPathOrParent(path string) (string, string, error) {
	// Ensure that the path is absolute and cleaned so that filesystem root
	// detection works.
	path, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("unable to compute absolute path: %w", err)
	}

	// Loop until a match is found or the filesystem root is reached.
	for {
		// Check for matches at this level.
		for _, name := range defaultConfigurationFileNames {
			if _, err := os.Stat(filepath.Join(path, name)); err == nil {
				return path, name, nil
			}
		}

		// Compute the parent directory. If we're already at the filesystem
		// root, then there's nowhere left to search.
		if parent := filepath.Dir(path); parent == path {
			return "", "", os.ErrNotExist
		} else {
			path = parent
		}
	}
}

// findConfigurationOverrideFileInPath searches the target path for a default
// Docker Compose configuration override file and returns the matching file
// name. It will return an error if multiple override files exist and
// os.ErrNotExist if no match is found. This function roughly models the logic
// of the get_default_override_file function in Docker Compose.
func findConfigurationOverrideFileInPath(path string) (string, error) {
	// Perform the search and watch for multiple matches.
	var result string
	for _, name := range defaultConfigurationOverrideFileNames {
		if _, err := os.Stat(filepath.Join(path, name)); err == nil {
			if result != "" {
				return "", errors.New("multiple configuration override files found")
			}
			result = name
		}
	}

	// Handle the case of no match.
	if result == "" {
		return "", os.ErrNotExist
	}

	// Success.
	return result, nil
}

// loadEnvironment loads a "dotenv" environment variable file from disk and
// merges in the content from the current environment (with the current
// environment taking precedence). If the target file doesn't exist, then it is
// treated as empty and the resulting environment will be the current process'
// environment.
func loadEnvironment(path string) (map[string]string, error) {
	// Create an empty (but initialized) environment.
	environment := make(map[string]string)

	// Load the environment file (if it exists) and add its contents. It's worth
	// noting that the godotenv package supports interpolation by default, which
	// is what Docker Compose uses by default when loading environment variable
	// files from disk.
	fileEnvironment, err := godotenv.Read(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to load environment file (%s): %w", path, err)
	}
	for key, value := range fileEnvironment {
		environment[key] = value
	}

	// Add environment variables from the OS.
	for _, specification := range os.Environ() {
		keyValue := strings.SplitN(specification, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid OS environment variable specification: %s", specification)
		}
		environment[keyValue[0]] = keyValue[1]
	}

	// Success.
	return environment, nil
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

// project contains the Docker Compose project metadata computed using top-level
// command line flags, environment variables, and emulated default behavior. It
// is initialized using initializeProject.
var project struct {
	// environment is the effective environment for the project.
	environment map[string]string
	// files are the configuration files for the project. If this field is nil,
	// then configuration should be read from standard input. Each path in this
	// slice will be an absolute and cleaned path, though paths may contain
	// symbolic link elements.
	files []string
	// workingDirectory is the working directory for the project. This is
	// computed using a combination of the current working directory, the
	// --project-directory flag, the -f/--file flag(s), and Docker Compose's
	// default resolution behavior. This value will be an absolute and cleaned
	// path, though it may contain symbolic link elements.
	workingDirectory string
	// name is the project name. This is computed using the values of the final
	// project working directory, the COMPOSE_PROJECT_NAME environment variable,
	// and the -p/--project-name flag.
	name string
}

// initializeProject initializes the project structure by partially emulating
// Docker Compose's project loading behavior. This isn't particularly expensive,
// but it does require some computation and filesystem scanning, so it isn't
// done by default. This function should only be called once. This function
// roughly models the logic of the project_from_options function in Docker
// Compose.
func initializeProject() error {
	// Compute the effective environment for the project by loading any "dotenv"
	// environment variable file from disk and then overriding its contents with
	// that of the current process' environment. Environment variable files are
	// always loaded relative to the current working directory (as opposed to
	// the final resolved project working directory), although they will be
	// loaded relative to the --project-directory path, if specified.
	environmentFilePath := rootConfiguration.envFile
	if environmentFilePath == "" {
		environmentFilePath = ".env"
	}
	if rootConfiguration.projectDirectory != "" {
		environmentFilePath = filepath.Join(
			rootConfiguration.projectDirectory,
			environmentFilePath,
		)
	}
	var err error
	project.environment, err = loadEnvironment(environmentFilePath)
	if err != nil {
		return fmt.Errorf("unable to compute environment: %w", err)
	}

	// Determine the configuration file specifications. This isn't the same as
	// determining the final configuration file paths, we're just determining
	// where we should look for specifications (i.e. on the command line or in
	// the environment) and the value of those specifications. There may also
	// not be any specifications (indicating that default search behavior should
	// be used). This code roughly models the logic of the
	// get_config_path_from_options function in Docker Compose.
	var configurationFiles []string
	if len(rootConfiguration.file) > 0 {
		configurationFiles = rootConfiguration.file
	} else if composeFile := project.environment["COMPOSE_FILE"]; composeFile != "" {
		separator, ok := project.environment["COMPOSE_PATH_SEPARATOR"]
		if !ok {
			separator = string(os.PathListSeparator)
		} else if separator == "" {
			return errors.New("empty separator specified by COMPOSE_PATH_SEPARATOR")
		}
		configurationFiles = strings.Split(composeFile, separator)
	}

	// If a project directory has been explicitly specified, then convert it to
	// an absolute path.
	var absoluteProjectDirectory string
	if rootConfiguration.projectDirectory != "" {
		absoluteProjectDirectory, err = filepath.Abs(rootConfiguration.projectDirectory)
		if err != nil {
			return fmt.Errorf(
				"unable to convert specified working directory (%s) to absolute path: %w",
				rootConfiguration.projectDirectory,
				err,
			)
		}
	}

	// Using the configuration file specifications, determine the final
	// configuration file paths and the working directory. The three scenarios
	// we need to handle are configuration read from standard input, explicitly
	// specified configuration files, and default configuration file searching
	// behavior. This code roughly models the logic of the config.find function
	// in Docker Compose.
	if len(configurationFiles) == 1 && configurationFiles[0] == "-" {
		if absoluteProjectDirectory != "" {
			project.workingDirectory = absoluteProjectDirectory
		} else {
			if project.workingDirectory, err = os.Getwd(); err != nil {
				return fmt.Errorf("unable to determine current working directory: %w", err)
			}
		}
	} else if len(configurationFiles) > 0 {
		project.files = make([]string, len(configurationFiles))
		for f, file := range configurationFiles {
			if absFile, err := filepath.Abs(file); err != nil {
				return fmt.Errorf("unable to convert file specification (%s) to absolute path: %w", file, err)
			} else {
				project.files[f] = absFile
			}
		}
		if absoluteProjectDirectory != "" {
			project.workingDirectory = absoluteProjectDirectory
		} else {
			project.workingDirectory = filepath.Dir(project.files[0])
		}
	} else {
		path, name, err := findConfigurationFileInPathOrParent(".")
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New("unable to identify configuration file in current directory or any parent")
			}
			return fmt.Errorf("unable to search for Docker Compose configuration file: %w", err)
		}
		project.files = append(project.files, filepath.Join(path, name))
		if overrideName, err := findConfigurationOverrideFileInPath(path); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("unable to identify configuration override file: %w", err)
			}
		} else {
			project.files = append(project.files, filepath.Join(path, overrideName))
		}
		if absoluteProjectDirectory != "" {
			project.workingDirectory = absoluteProjectDirectory
		} else {
			project.workingDirectory = path
		}
	}

	// Determine the project name. This code roughly models the logic of the
	// get_project_name function in Docker Compose.
	if rootConfiguration.projectName != "" {
		project.name = normalizeProjectName(rootConfiguration.projectName)
	} else if projectName := project.environment["COMPOSE_PROJECT_NAME"]; projectName != "" {
		project.name = normalizeProjectName(projectName)
	} else if projectName = filepath.Base(project.workingDirectory); projectName != "" {
		project.name = normalizeProjectName(projectName)
	} else {
		project.name = "default"
	}

	// Success.
	return nil
}

// compose invokes Docker Compose with the specified arguments, environment,
// standard input, and exit behavior. If environment is nil, then the Docker
// Compose process will inherit the current environment. If input is nil, then
// Docker Compose will read from the null device (os.DevNull). If an error
// occurs while attempting to invoke Docker Compose, then this function will
// print an error message and terminate the current process with an exit code of
// 1. If invocation succeeds but Docker Compose exits with a non-0 exit code,
// then this function won't print an error message but will terminate the
// current process with a matching exit code. If invocation succeeds and Docker
// Compose exits with an exit code of 0, then this function will simply return,
// unless exitOnSuccess is specified, in which case this process will terminate
// the current process with an exit code of 0.
func compose(arguments []string, environment map[string]string, input io.Reader, exitOnSuccess bool) {
	// Create the command.
	compose := exec.Command("docker-compose", arguments...)

	// Set up the command environment.
	if environment != nil {
		compose.Env = make([]string, 0, len(environment))
		for k, v := range environment {
			compose.Env = append(compose.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Set up input and output streams.
	compose.Stdin = input
	compose.Stdout = os.Stdout
	compose.Stderr = os.Stderr

	// Run the command.
	if err := compose.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitCode := exitErr.ExitCode(); exitCode < 1 {
				os.Exit(1)
			} else {
				os.Exit(exitCode)
			}
		} else {
			cmd.Fatal(fmt.Errorf("unable to invoke Docker Compose: %w", err))
		}
	}

	// Terminate the current process if necessary.
	if exitOnSuccess {
		os.Exit(0)
	}
}

// passthrough is a generic Cobra handler that will pass handling directly to
// Docker Compose using the command name, reconstituted top-level flags, and
// command arguments. In order to use this handler, flag parsing must be
// disabled for the command.
func passthrough(command *cobra.Command, arguments []string) {
	arguments = append([]string{command.CalledAs()}, arguments...)
	arguments = append(topLevelFlags(), arguments...)
	compose(arguments, nil, os.Stdin, true)
}
