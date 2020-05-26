package compose

// ProjectFlags encodes top-level Docker Compose command line flags that control
// the Docker Compose project. This type is designed to be used as command line
// flag storage. The zero value of this structure is a valid value corresponding
// to the absence of any of these flags.
type ProjectFlags struct {
	// File stores the value(s) of the -f/--file flag(s).
	File []string
	// ProjectName stores the value of the -p/--project-name flag.
	ProjectName string
	// ProjectDirectory stores the value of the --project-directory flag.
	ProjectDirectory string
	// EnvFile stores the value of the --env-file flag.
	EnvFile string
}
