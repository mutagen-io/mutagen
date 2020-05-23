package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
