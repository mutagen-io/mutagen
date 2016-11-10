package agent

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/howeyc/gopass"

	"github.com/havoc-io/mutagen/environment"
	"github.com/havoc-io/mutagen/process"
	"github.com/havoc-io/mutagen/url"
)

const (
	PrompterEnvironmentVariable              = "MUTAGEN_PROMPTER"
	PrompterContextBase64EnvironmentVariable = "MUTAGEN_PROMPTER_CONTEXT_BASE64"

	sshConnectTimeoutSeconds = 5
)

type PromptClass uint8

const (
	PromptClassSecret PromptClass = iota
	PromptClassDisplay
	PromptClassBinary
)

func ClassifyPrompt(prompt string) PromptClass {
	// TODO: Implement using white-listing regexes based on known OpenSSH
	// prompts.
	return PromptClassSecret
}

func PromptCommandLine(context, prompt string) (string, error) {
	// Classify the prompt.
	class := ClassifyPrompt(prompt)

	// Figure out which getter to use.
	var getter func() ([]byte, error)
	if class == PromptClassDisplay || class == PromptClassBinary {
		getter = gopass.GetPasswdEchoed
	} else {
		getter = gopass.GetPasswd
	}

	// Print the context (if any) and the prompt.
	if context != "" {
		fmt.Println(context)
	}
	fmt.Print(prompt)

	// Get the result.
	result, err := getter()
	if err != nil {
		return "", errors.Wrap(err, "unable to read response")
	}

	// Success.
	return string(result), nil
}

func prompterEnvironment(prompter, context string) []string {
	// If there is no prompter, return nil to just use the current environment.
	if prompter == "" {
		return nil
	}

	// Convert context to base64 encoding so that we can pass it through the
	// environment safely.
	contextBase64 := base64.StdEncoding.EncodeToString([]byte(context))

	// Create a copy of the current environment.
	result := make(map[string]string, len(environment.Current))
	for k, v := range environment.Current {
		result[k] = v
	}

	// Insert necessary environment variables.
	result["SSH_ASKPASS"] = process.Current.ExecutablePath
	result["DISPLAY"] = "mutagen"
	result[PrompterEnvironmentVariable] = prompter
	result[PrompterContextBase64EnvironmentVariable] = contextBase64

	// Convert into the desired format.
	return environment.Format(result)
}

// TODO: Document that the local path must be absolute.
func scp(prompter, context, local string, remote *url.SSHURL) error {
	// Locate the SCP command.
	scp, err := scpCommand()
	if err != nil {
		return errors.Wrap(err, "unable to identify SCP executable")
	}

	// HACK: On Windows, we attempt to use SCP executables that might not
	// understand Windows paths because they're designed to run inside a POSIX-
	// style environment (e.g. MSYS or Cygwin). To work around this, we run them
	// in the same directory as the source and just pass them the source base
	// name. This works fine on other systems as well. Unfortunately this means
	// that we need to use absolute paths, but we do that anyway.
	if !filepath.IsAbs(local) {
		return errors.New("scp source path must be absolute")
	}
	workingDirectory, sourceBase := filepath.Split(local)

	// Compute the destination URL.
	destinationURL := fmt.Sprintf("%s:%s", remote.Hostname, remote.Path)
	if remote.Username != "" {
		destinationURL = fmt.Sprintf("%s@%s", remote.Username, destinationURL)
	}

	// Set up arguments.
	var scpArguments []string
	scpArguments = append(scpArguments, fmt.Sprintf("-oConnectTimeout=%d", sshConnectTimeoutSeconds))
	if remote.Port != 0 {
		scpArguments = append(scpArguments, "-P", fmt.Sprintf("%d", remote.Port))
	}
	scpArguments = append(scpArguments, sourceBase, destinationURL)

	// Create the process.
	scpProcess := exec.Command(scp, scpArguments...)

	// Set the working directory.
	scpProcess.Dir = workingDirectory

	// Force it to run detached.
	scpProcess.SysProcAttr = processAttributes()

	// Set the environment necessary for prompting.
	scpProcess.Env = prompterEnvironment(prompter, context)

	// Run the operation.
	if err = scpProcess.Run(); err != nil {
		return errors.Wrap(err, "unable to run SCP process")
	}

	// Success.
	return nil
}

// TODO: Document that the URL path is NOT used as a working directory, it is
// simply ignored.
func ssh(prompter, context string, remote *url.SSHURL, command string) (*exec.Cmd, error) {
	// Locate the SSH command.
	ssh, err := sshCommand()
	if err != nil {
		return nil, errors.Wrap(err, "unable to identify SSH executable")
	}

	// Compute the target.
	target := remote.Hostname
	if remote.Username != "" {
		target = fmt.Sprintf("%s@%s", remote.Username, remote.Hostname)
	}

	// Set up arguments.
	var sshArguments []string
	sshArguments = append(sshArguments, fmt.Sprintf("-oConnectTimeout=%d", sshConnectTimeoutSeconds))
	if remote.Port != 0 {
		sshArguments = append(sshArguments, "-p", fmt.Sprintf("%d", remote.Port))
	}
	sshArguments = append(sshArguments, target, command)

	// Create the process.
	sshProcess := exec.Command(ssh, sshArguments...)

	// Force it to run detached.
	sshProcess.SysProcAttr = processAttributes()

	// Set the environment necessary for prompting.
	sshProcess.Env = prompterEnvironment(prompter, context)

	// Done.
	return sshProcess, nil
}

func sshRun(prompter, context string, remote *url.SSHURL, command string) error {
	// Create the process.
	process, err := ssh(prompter, context, remote, command)
	if err != nil {
		return errors.Wrap(err, "unable to create command")
	}

	// Run the process.
	return process.Run()
}

func sshOutput(prompter, context string, remote *url.SSHURL, command string) ([]byte, error) {
	// Create the process.
	process, err := ssh(prompter, context, remote, command)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create command")
	}

	// Run the process.
	return process.Output()
}
