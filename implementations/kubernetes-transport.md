# Kubernetes Transport Layer Implementation Plan

## Overview

This document outlines the implementation of a new Kubernetes transport layer for Mutagen, enabling file synchronization and network forwarding to containers running in Kubernetes clusters via `kubectl`.

**Estimated Time:** 50-60 hours

**MVP Scope:** Linux container pods with shell (`/bin/sh`) and `tar` available, validated kubeconfig and context configuration. Windows containers are supported but marked as experimental.

## Architecture Summary

The Kubernetes transport will follow the existing Docker transport pattern:
- **URL Protocol**: `kubernetes://`
- **Transport**: Uses `kubectl exec` and `kubectl cp` commands
- **Agent Deployment**: Copies the Mutagen agent binary into the pod and executes it

---

## Components to Implement

### Phase 1: Core Infrastructure

#### 1.1 Protocol Definition (`pkg/url/url.proto`)

Add a new protocol enum value:

```protobuf
// Kubernetes indicates that the resource is inside a Kubernetes pod.
Kubernetes = 12;
```

**Files to modify:**
- `pkg/url/url.proto` - Add `Kubernetes = 12` to the Protocol enum
- Run `go generate ./pkg/...` to regenerate `url.pb.go`

#### 1.2 URL Parsing (`pkg/url/parse_kubernetes.go`)

Create a new file to parse Kubernetes URLs.

**Simplified URL Format (context via flags/parameters only):**

```
kubernetes://[user@]namespace/pod[/container]:/path
kubernetes://[user@]namespace/pod[/container]:tcp:host:port
```

**Design Decision:** Context is intentionally excluded from the URL host component to avoid ambiguity with the namespace/pod/container hierarchy. Context must be specified via:
- CLI flag: `--kubernetes-context`
- URL parameter (persisted internally)
- Environment variable: `KUBECONFIG` (which may contain context)

**Examples:**
- `kubernetes://default/my-pod:/app/data` - Pod `my-pod` in namespace `default`
- `kubernetes://prod/my-pod/main:/data` - Container `main` in pod `my-pod` in namespace `prod`
- `kubernetes://user@default/my-pod:/home/user/data` - Run as specific user (note: user is validated but not enforced by kubectl; used for ownership)

**Environment variables to capture:**
- `KUBECONFIG` - Path(s) to kubeconfig file(s)

**URL Parameters (for command-line flag persistence):**
- `context` - Kubernetes context (from `--kubernetes-context`)
- `namespace` - Kubernetes namespace (can override URL namespace)
- `container` - Container name (can override URL container)
- `kubeconfig` - Path to kubeconfig file

**Parsing Implementation:**

```go
// pkg/url/parse_kubernetes.go
package url

import (
    "errors"
    "fmt"
    "strings"

    "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// kubernetesURLPrefix is the lowercase version of the Kubernetes URL prefix.
const kubernetesURLPrefix = "kubernetes://"

// KubernetesEnvironmentVariables is a list of Kubernetes environment variables
// that should be locked in to Kubernetes URLs at parse time.
var KubernetesEnvironmentVariables = []string{
    "KUBECONFIG",
}

// kubernetesParameterNames is a list of supported Kubernetes URL parameters.
var kubernetesParameterNames = []string{
    "kubeconfig",
    "context",
    "namespace",
    "container",
}

// isKubernetesURL checks whether or not a URL is a Kubernetes URL.
func isKubernetesURL(raw string) bool {
    return strings.HasPrefix(strings.ToLower(raw), kubernetesURLPrefix)
}

// ParsedKubernetesHost contains the parsed components of a Kubernetes URL host.
type ParsedKubernetesHost struct {
    Namespace string
    Pod       string
    Container string
}

// ParseKubernetesHost parses a Kubernetes URL host component into its parts.
// The host format is: namespace/pod[/container]
// Returns an error if the format is invalid.
func ParseKubernetesHost(host string) (*ParsedKubernetesHost, error) {
    parts := strings.Split(host, "/")
    
    switch len(parts) {
    case 2:
        // namespace/pod
        if parts[0] == "" {
            return nil, errors.New("empty namespace")
        }
        if parts[1] == "" {
            return nil, errors.New("empty pod name")
        }
        return &ParsedKubernetesHost{
            Namespace: parts[0],
            Pod:       parts[1],
        }, nil
    case 3:
        // namespace/pod/container
        if parts[0] == "" {
            return nil, errors.New("empty namespace")
        }
        if parts[1] == "" {
            return nil, errors.New("empty pod name")
        }
        if parts[2] == "" {
            return nil, errors.New("empty container name")
        }
        return &ParsedKubernetesHost{
            Namespace: parts[0],
            Pod:       parts[1],
            Container: parts[2],
        }, nil
    case 1:
        // Just pod name - require namespace
        return nil, errors.New("namespace is required (format: namespace/pod[/container])")
    default:
        return nil, fmt.Errorf("invalid host format: expected namespace/pod[/container], got %d components", len(parts))
    }
}

// parseKubernetes parses a Kubernetes URL.
func parseKubernetes(raw string, kind Kind, first bool) (*URL, error) {
    // Strip off the prefix.
    raw = raw[len(kubernetesURLPrefix):]

    // Determine the character that splits the host from the path or
    // forwarding endpoint component.
    var splitCharacter rune
    if kind == Kind_Synchronization {
        splitCharacter = ':'
    } else if kind == Kind_Forwarding {
        splitCharacter = ':'
    } else {
        panic("unhandled URL kind")
    }

    // Parse off the username. If we hit a '/', then we've reached part of the
    // host specification and there was no username. Similarly, if we hit ':',
    // we've reached the path delimiter.
    var username string
    for i, r := range raw {
        if r == '/' || r == splitCharacter {
            break
        } else if r == '@' {
            username = raw[:i]
            raw = raw[i+1:]
            break
        }
    }

    // Split what remains into the host and the path (or forwarding endpoint).
    // For Kubernetes, the host contains slashes (namespace/pod/container),
    // so we need to find the first ':' that separates host from path.
    var host, path string
    colonIndex := strings.Index(raw, ":")
    if colonIndex == -1 {
        return nil, errors.New("missing path separator ':'")
    }
    host = raw[:colonIndex]
    path = raw[colonIndex+1:]

    // Validate the host format.
    if _, err := ParseKubernetesHost(host); err != nil {
        return nil, fmt.Errorf("invalid host: %w", err)
    }

    // Validate path based on URL kind.
    if kind == Kind_Synchronization {
        if path == "" {
            return nil, errors.New("missing path")
        }
        // Handle home-directory-relative paths.
        if strings.HasPrefix(path, "~") {
            // Keep as-is, will be resolved relative to home directory
        } else if !strings.HasPrefix(path, "/") && !isWindowsPath(path) {
            // Relative path - treat as relative to home directory
            path = "~/" + path
        }
    } else if kind == Kind_Forwarding {
        if path == "" {
            return nil, errors.New("missing forwarding endpoint")
        }
        // Parse the forwarding endpoint URL to ensure that it's valid.
        if _, _, err := forwarding.Parse(path); err != nil {
            return nil, fmt.Errorf("invalid forwarding endpoint URL: %w", err)
        }
    } else {
        panic("unhandled URL kind")
    }

    // Store any Kubernetes environment variables that we need to preserve.
    environment := make(map[string]string)
    for _, variable := range KubernetesEnvironmentVariables {
        if value, present := getEnvironmentVariable(variable, kind, first); present {
            environment[variable] = value
        }
    }

    // Success.
    return &URL{
        Kind:        kind,
        Protocol:    Protocol_Kubernetes,
        User:        username,
        Host:        host,
        Path:        path,
        Environment: environment,
    }, nil
}
```

#### 1.3 URL Validation & Formatting

**Files to modify:**
- `pkg/url/url.go` - Add `Protocol_Kubernetes` to validation and formatting logic
- `pkg/url/format.go` - Add formatting support for Kubernetes URLs
- `pkg/url/parse.go` - Add Kubernetes URL detection and dispatch

---

### Phase 2: Kubectl Package

#### 2.1 Kubectl Command Wrapper (`pkg/kubernetes/kubernetes.go`)

Create a new package similar to `pkg/docker/`:

```go
// pkg/kubernetes/kubernetes.go
package kubernetes

import (
    "context"
    "fmt"
    "os"
    "os/exec"

    "github.com/mutagen-io/mutagen/pkg/platform"
)

// CommandPath returns the absolute path specification to use for invoking
// kubectl. It will use the MUTAGEN_KUBECTL_PATH environment variable if
// provided, otherwise falling back to a platform-specific implementation.
func CommandPath() (string, error) {
    // If MUTAGEN_KUBECTL_PATH is specified, then use it to perform the lookup.
    if searchPath := os.Getenv("MUTAGEN_KUBECTL_PATH"); searchPath != "" {
        return platform.FindCommand("kubectl", []string{searchPath})
    }

    // Otherwise fall back to the platform-specific implementation.
    return commandPathForPlatform()
}

// Command prepares (but does not start) a kubectl command with the specified
// arguments and scoped to lifetime of the provided context.
func Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
    // Identify the command path.
    commandPath, err := CommandPath()
    if err != nil {
        return nil, fmt.Errorf("unable to identify 'kubectl' command: %w", err)
    }

    // Create the command.
    return exec.CommandContext(ctx, commandPath, args...), nil
}
```

**Files to create:**
- `pkg/kubernetes/doc.go`
- `pkg/kubernetes/kubernetes.go`
- `pkg/kubernetes/kubernetes_darwin.go`
- `pkg/kubernetes/kubernetes_posix.go`
- `pkg/kubernetes/kubernetes_windows.go`
- `pkg/kubernetes/kubernetes_test.go`

#### 2.2 Connection Flags (`pkg/kubernetes/flags.go`)

```go
// pkg/kubernetes/flags.go
package kubernetes

import (
    "errors"
    "fmt"
)

// ConnectionFlags encodes kubectl command line flags that control the
// Kubernetes cluster connection. These flags can be loaded from Mutagen URL
// parameters or used as command line flag storage.
type ConnectionFlags struct {
    // Kubeconfig stores the value of the --kubeconfig flag.
    Kubeconfig string
    // Context stores the value of the --context flag.
    Context string
    // Namespace stores the value of the -n/--namespace flag.
    Namespace string
    // Container stores the value of the -c/--container flag.
    Container string
}

// LoadConnectionFlagsFromURLParameters loads kubectl connection flags from
// Mutagen URL parameters.
func LoadConnectionFlagsFromURLParameters(parameters map[string]string) (*ConnectionFlags, error) {
    result := &ConnectionFlags{}

    for key, value := range parameters {
        switch key {
        case "kubeconfig":
            if value == "" {
                return nil, errors.New("kubeconfig parameter has empty value")
            }
            result.Kubeconfig = value
        case "context":
            if value == "" {
                return nil, errors.New("context parameter has empty value")
            }
            result.Context = value
        case "namespace":
            if value == "" {
                return nil, errors.New("namespace parameter has empty value")
            }
            result.Namespace = value
        case "container":
            if value == "" {
                return nil, errors.New("container parameter has empty value")
            }
            result.Container = value
        default:
            return nil, fmt.Errorf("unknown parameter: %s", key)
        }
    }

    return result, nil
}

// ToFlags reconstitutes connection flags for passing to kubectl commands.
// Note: Container flag is NOT included here as it's command-specific
// (used differently by exec vs cp).
func (f *ConnectionFlags) ToFlags() []string {
    var result []string

    if f.Kubeconfig != "" {
        result = append(result, "--kubeconfig", f.Kubeconfig)
    }
    if f.Context != "" {
        result = append(result, "--context", f.Context)
    }
    if f.Namespace != "" {
        result = append(result, "-n", f.Namespace)
    }

    return result
}

// ToURLParameters converts connection flags to URL parameters.
func (f *ConnectionFlags) ToURLParameters() map[string]string {
    result := make(map[string]string)

    if f.Kubeconfig != "" {
        result["kubeconfig"] = f.Kubeconfig
    }
    if f.Context != "" {
        result["context"] = f.Context
    }
    if f.Namespace != "" {
        result["namespace"] = f.Namespace
    }
    if f.Container != "" {
        result["container"] = f.Container
    }

    return result
}
```

**Files to create:**
- `pkg/kubernetes/flags.go`
- `pkg/kubernetes/flags_test.go`

---

### Phase 3: Transport Implementation

#### 3.1 Kubernetes Transport (`pkg/agent/transport/kubernetes/`)

**Files to create:**
- `pkg/agent/transport/kubernetes/doc.go`
- `pkg/agent/transport/kubernetes/transport.go`
- `pkg/agent/transport/kubernetes/transport_test.go`
- `pkg/agent/transport/kubernetes/environment.go`

**Transport Structure:**

```go
// pkg/agent/transport/kubernetes/transport.go
package kubernetes

import (
    "context"
    "errors"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "unicode/utf8"

    "github.com/mutagen-io/mutagen/pkg/agent"
    "github.com/mutagen-io/mutagen/pkg/agent/transport"
    "github.com/mutagen-io/mutagen/pkg/environment"
    "github.com/mutagen-io/mutagen/pkg/kubernetes"
    "github.com/mutagen-io/mutagen/pkg/process"
)

// Error fragment detection constants. Multiple fragments are checked for each
// error type to handle variation across kubectl versions.
var (
    // podNotFoundFragments are fragments indicating the pod does not exist.
    podNotFoundFragments = []string{"not found", "NotFound", "doesn't exist"}
    // podNotRunningFragments are fragments indicating the pod is not running.
    podNotRunningFragments = []string{"is not running", "not running", "ContainerCreating", "Pending", "Terminated", "CrashLoopBackOff"}
    // containerNotFoundFragments are fragments indicating the container does not exist.
    containerNotFoundFragments = []string{"container not found", "container \"", "Invalid container"}
    // multiContainerFragments indicate multiple containers exist without specification.
    multiContainerFragments = []string{"must specify a container", "Defaulting container", "has multiple containers"}
    // forbiddenFragments indicate an authorization error.
    forbiddenFragments = []string{"forbidden", "Forbidden", "unauthorized", "Unauthorized", "RBAC"}
    // namespaceNotFoundFragments indicate the namespace does not exist.
    namespaceNotFoundFragments = []string{"namespace", "namespaces", "not found", "NotFound"}
    // contextNotFoundFragments indicate the Kubernetes context is invalid.
    contextNotFoundFragments = []string{"context", "not found", "does not exist", "no context"}
    // kubeconfigNotFoundFragments indicate kubeconfig file issues.
    kubeconfigNotFoundFragments = []string{"kubeconfig", "no such file", "unable to load", "invalid configuration"}
    // clusterNotReachableFragments indicate the cluster cannot be contacted.
    clusterNotReachableFragments = []string{"connection refused", "no route to host", "timeout", "unable to connect", "dial tcp", "i/o timeout", "deadline exceeded"}
    // tarNotFoundFragments indicate tar is not available in the container.
    tarNotFoundFragments = []string{"tar: not found", "tar: command not found", "executable file not found", "OCI runtime exec failed"}
    // readOnlyFilesystemFragments indicate a read-only filesystem error.
    readOnlyFilesystemFragments = []string{"read-only file system", "Read-only file system", "EROFS"}
    // podRestartedFragments indicate the pod or container has been restarted.
    podRestartedFragments = []string{"container has been restarted", "container exited", "has been deleted"}
)

// isContextOrClusterError checks if the error is related to context or cluster
// connectivity rather than pod/container issues.
func isContextOrClusterError(message string) bool {
    messageLower := strings.ToLower(message)
    // Context errors contain "context" and an error indicator.
    if strings.Contains(messageLower, "context") &&
        (strings.Contains(messageLower, "not found") ||
            strings.Contains(messageLower, "does not exist") ||
            strings.Contains(messageLower, "no context") ||
            strings.Contains(messageLower, "is not set") ||
            strings.Contains(messageLower, "not configured") ||
            strings.Contains(messageLower, "invalid")) {
        return true
    }
    // Check for "current-context is not set" specifically.
    if strings.Contains(messageLower, "current-context") {
        return true
    }
    // Cluster connectivity errors.
    for _, fragment := range clusterNotReachableFragments {
        if strings.Contains(messageLower, strings.ToLower(fragment)) {
            return true
        }
    }
    return false
}

// isNamespaceError checks if the error indicates a namespace issue.
// Namespace errors contain both "namespace" and "not found".
func isNamespaceError(message string) bool {
    messageLower := strings.ToLower(message)
    return (strings.Contains(messageLower, "namespace") || strings.Contains(messageLower, "namespaces")) &&
        (strings.Contains(messageLower, "not found") || strings.Contains(messageLower, "notfound"))
}

// containsAnyFragment checks if the message contains any of the given fragments.
func containsAnyFragment(message string, fragments []string) bool {
    messageLower := strings.ToLower(message)
    for _, fragment := range fragments {
        if strings.Contains(messageLower, strings.ToLower(fragment)) {
            return true
        }
    }
    return false
}

// kubernetesTransport implements the agent.Transport interface using kubectl.
type kubernetesTransport struct {
    // pod is the target pod name.
    pod string
    // namespace is the target namespace (from URL host, may be overridden).
    namespace string
    // container is the target container name within the pod.
    container string
    // user is the user specified in the URL. This is used for validation and
    // ownership purposes, not for kubectl authentication (which doesn't support
    // per-command user switching).
    user string
    // environment is the collection of environment variables that need to be
    // set for the kubectl executable (e.g., KUBECONFIG).
    environment map[string]string
    // kubeconfig is the path to the kubeconfig file (from parameters).
    kubeconfig string
    // kubeContext is the Kubernetes context to use (from parameters).
    kubeContext string
    // effectiveNamespace is the resolved namespace (parameters override URL).
    effectiveNamespace string
    // effectiveContainer is the resolved container (parameters override URL).
    effectiveContainer string
    // prompter is the prompter identifier to use for prompting.
    prompter string
    // containerProbed indicates whether or not container probing has occurred.
    containerProbed bool
    // containerIsWindows indicates whether or not the container is Windows.
    containerIsWindows bool
    // containerHomeDirectory is the path to the user's home directory.
    containerHomeDirectory string
    // containerUser is the name of the user inside the container.
    containerUser string
    // containerUserGroup is the default group for the user.
    containerUserGroup string
    // containerProbeError tracks any error that arose when probing.
    containerProbeError error
    // containerHasShell indicates whether the container has a shell available.
    // This is used to determine if we can wrap commands for working directory support.
    containerHasShell bool
}

// NewTransport creates a new Kubernetes transport using the specified parameters.
func NewTransport(
    pod, namespace, container, user string,
    env, parameters map[string]string,
    prompter string,
) (agent.Transport, error) {
    // Load connection flags from URL parameters.
    connectionFlags, err := kubernetes.LoadConnectionFlagsFromURLParameters(parameters)
    if err != nil {
        return nil, fmt.Errorf("unable to compute connection flags: %w", err)
    }

    // Resolve effective namespace: parameters override URL.
    effectiveNamespace := namespace
    if connectionFlags.Namespace != "" {
        effectiveNamespace = connectionFlags.Namespace
    }
    if effectiveNamespace == "" {
        return nil, errors.New("namespace is required")
    }

    // Resolve effective container: parameters override URL.
    effectiveContainer := container
    if connectionFlags.Container != "" {
        effectiveContainer = connectionFlags.Container
    }

    // Validate pod name.
    if pod == "" {
        return nil, errors.New("pod name is required")
    }

    return &kubernetesTransport{
        pod:                pod,
        namespace:          namespace,
        container:          container,
        user:               user,
        environment:        env,
        kubeconfig:         connectionFlags.Kubeconfig,
        kubeContext:        connectionFlags.Context,
        effectiveNamespace: effectiveNamespace,
        effectiveContainer: effectiveContainer,
        prompter:           prompter,
    }, nil
}

// buildBaseFlags constructs the common kubectl flags for cluster connection.
// These flags come BEFORE the subcommand (exec, cp, etc.).
func (t *kubernetesTransport) buildBaseFlags() []string {
    var flags []string

    if t.kubeconfig != "" {
        flags = append(flags, "--kubeconfig", t.kubeconfig)
    }
    if t.kubeContext != "" {
        flags = append(flags, "--context", t.kubeContext)
    }
    if t.effectiveNamespace != "" {
        flags = append(flags, "-n", t.effectiveNamespace)
    }

    return flags
}

// shellEscape escapes a string for safe use in shell commands.
// This handles spaces, special characters, and prevents injection attacks.
func shellEscape(s string) string {
    // If the string contains no special characters, return as-is.
    if !strings.ContainsAny(s, " \t\n'\"\\$`!*?[]{}()&|;<>#~") {
        return s
    }
    // Use single quotes and escape any embedded single quotes.
    // 'path with spaces' -> 'path with spaces'
    // path's -> 'path'\''s'
    return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// windowsEscape escapes a string for safe use in Windows cmd.exe commands.
func windowsEscape(s string) string {
    // If the string contains spaces or special characters, wrap in quotes.
    if strings.ContainsAny(s, " \t&|<>^") {
        // Double up any existing double quotes and wrap.
        escaped := strings.ReplaceAll(s, "\"", "\"\"")
        return "\"" + escaped + "\""
    }
    return s
}

// command creates a kubectl exec command with proper working directory support.
// If the container lacks a shell (distroless, busybox without sh, etc.) and a
// working directory is specified, this method will return an error suggesting
// the user ensure the container has a shell or use a different base image.
func (t *kubernetesTransport) command(cmd, workingDirectory string) (*exec.Cmd, error) {
    // Build kubectl arguments.
    var kubectlArguments []string

    // Add base connection flags FIRST (before subcommand).
    kubectlArguments = append(kubectlArguments, t.buildBaseFlags()...)

    // Add exec subcommand with interactive flag.
    kubectlArguments = append(kubectlArguments, "exec", "-i")

    // Add container specification if provided (BEFORE pod name).
    if t.effectiveContainer != "" {
        kubectlArguments = append(kubectlArguments, "-c", t.effectiveContainer)
    }

    // Add the pod name.
    kubectlArguments = append(kubectlArguments, t.pod)

    // Add command separator.
    kubectlArguments = append(kubectlArguments, "--")

    // Handle working directory requirement.
    // kubectl exec doesn't support --workdir, so we need to wrap with shell.
    if workingDirectory != "" && !t.containerHasShell && !t.containerIsWindows {
        // Container lacks a shell and we need working directory support.
        // The agent binary MUST be executed from the home directory.
        // Since the Copy puts files in home and Command uses home as workdir,
        // and the agent is invoked directly, this should be acceptable.
        // However, if workingDirectory differs from home, we have a problem.
        if workingDirectory != t.containerHomeDirectory {
            return nil, fmt.Errorf(
                "container does not have a shell (/bin/sh) and cannot change working directory; "+
                    "either ensure the container has a shell or use a different image; "+
                    "working directory %q cannot be used",
                workingDirectory,
            )
        }
        // If workingDirectory equals home, we can skip the cd wrapper since
        // kubectl exec runs in the container's default directory (usually /).
        // The agent handles relative paths from its location.
    }

    // Build the command to execute.
    if t.containerIsWindows {
        // Windows: use cmd /c for shell wrapping.
        // Escape the working directory path for Windows cmd.exe.
        var shellCommand string
        if workingDirectory != "" {
            shellCommand = fmt.Sprintf("cd /d %s && %s", windowsEscape(workingDirectory), cmd)
        } else {
            shellCommand = cmd
        }
        kubectlArguments = append(kubectlArguments, "cmd", "/c", shellCommand)
    } else if t.containerHasShell {
        // POSIX with shell: wrap in sh -c to handle cd && command.
        // Escape the working directory path for POSIX shell.
        var shellCommand string
        if workingDirectory != "" {
            shellCommand = fmt.Sprintf("cd %s && %s", shellEscape(workingDirectory), cmd)
        } else {
            shellCommand = cmd
        }
        kubectlArguments = append(kubectlArguments, "sh", "-c", shellCommand)
    } else {
        // POSIX without shell: execute command directly.
        // This works for the agent binary since it's an absolute path.
        // Split cmd into arguments (simple split, agent commands are simple).
        cmdParts := strings.Fields(cmd)
        kubectlArguments = append(kubectlArguments, cmdParts...)
    }

    // Create the command.
    kubectlCommand, err := kubernetes.Command(context.Background(), kubectlArguments...)
    if err != nil {
        return nil, err
    }

    // Set process attributes.
    kubectlCommand.SysProcAttr = transport.ProcessAttributes()

    // Build environment, filtering and setting Kubernetes variables.
    cmdEnv := kubectlCommand.Environ()
    cmdEnv = setKubernetesVariables(cmdEnv, t.environment)
    kubectlCommand.Env = cmdEnv

    return kubectlCommand, nil
}

// commandDirect creates a kubectl exec command for probing (no shell wrapper).
// This is used during probing when we don't yet know the container type.
func (t *kubernetesTransport) commandDirect(args ...string) (*exec.Cmd, error) {
    var kubectlArguments []string

    // Add base connection flags.
    kubectlArguments = append(kubectlArguments, t.buildBaseFlags()...)

    // Add exec subcommand with interactive flag.
    kubectlArguments = append(kubectlArguments, "exec", "-i")

    // Add container specification if provided.
    if t.effectiveContainer != "" {
        kubectlArguments = append(kubectlArguments, "-c", t.effectiveContainer)
    }

    // Add the pod name.
    kubectlArguments = append(kubectlArguments, t.pod)

    // Add command separator and the command.
    kubectlArguments = append(kubectlArguments, "--")
    kubectlArguments = append(kubectlArguments, args...)

    // Create the command.
    kubectlCommand, err := kubernetes.Command(context.Background(), kubectlArguments...)
    if err != nil {
        return nil, err
    }

    // Set process attributes.
    kubectlCommand.SysProcAttr = transport.ProcessAttributes()

    // Build environment.
    cmdEnv := kubectlCommand.Environ()
    cmdEnv = setKubernetesVariables(cmdEnv, t.environment)
    kubectlCommand.Env = cmdEnv

    return kubectlCommand, nil
}

// findEnvironmentVariable parses an environment variable block and searches
// for the specified variable. This follows the Docker transport's pattern.
func findEnvironmentVariable(block, variable string) (string, bool) {
    // Parse the environment variable block.
    parsed := environment.ParseBlock(block)

    // Search through the environment for the specified variable.
    for _, line := range parsed {
        if strings.HasPrefix(line, variable+"=") {
            return line[len(variable)+1:], true
        }
    }

    // No match.
    return "", false
}

// probeContainer ensures that container properties have been probed.
// This follows the Docker transport's probing logic closely for parity.
func (t *kubernetesTransport) probeContainer() error {
    // Watch for previous errors.
    if t.containerProbeError != nil {
        return fmt.Errorf("previous container probing failed: %w", t.containerProbeError)
    }

    // Check if already probed.
    if t.containerProbed {
        return nil
    }
    t.containerProbed = true

    // Track what we've discovered in probes.
    var windows bool
    var home string
    var posixEnv string
    var posixErr, windowsErr error
    var hasShell bool = true // Assume shell exists until proven otherwise

    // Attempt to run env in the container to probe the user's environment on
    // POSIX systems and identify the HOME environment variable value.
    if command, err := t.commandDirect("env"); err != nil {
        return fmt.Errorf("unable to set up kubectl invocation: %w", err)
    } else if envBytes, err := command.Output(); err != nil {
        if message := process.ExtractExitErrorMessage(err); message != "" {
            // Check for cluster/context/kubeconfig errors first - these indicate
            // configuration issues rather than pod/container problems.
            // Precedence: context/cluster → kubeconfig → namespace → pod → container → forbidden
            if isContextOrClusterError(message) {
                if strings.Contains(strings.ToLower(message), "context") {
                    t.containerProbeError = errors.New("Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG")
                } else {
                    t.containerProbeError = errors.New("unable to connect to Kubernetes cluster; check cluster connectivity and credentials")
                }
                return t.containerProbeError
            } else if containsAnyFragment(message, kubeconfigNotFoundFragments) {
                t.containerProbeError = errors.New("kubeconfig file not found or invalid; check --kubernetes-kubeconfig or KUBECONFIG environment variable")
                return t.containerProbeError
            } else if isNamespaceError(message) {
                t.containerProbeError = fmt.Errorf("namespace %q not found", t.effectiveNamespace)
                return t.containerProbeError
            } else if containsAnyFragment(message, podNotFoundFragments) && !isNamespaceError(message) {
                t.containerProbeError = errors.New("pod does not exist")
                return t.containerProbeError
            } else if containsAnyFragment(message, podNotRunningFragments) {
                t.containerProbeError = errors.New("pod is not running")
                return t.containerProbeError
            } else if containsAnyFragment(message, containerNotFoundFragments) {
                t.containerProbeError = fmt.Errorf("container %q not found in pod", t.effectiveContainer)
                return t.containerProbeError
            } else if containsAnyFragment(message, multiContainerFragments) {
                t.containerProbeError = errors.New("pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)")
                return t.containerProbeError
            } else if containsAnyFragment(message, forbiddenFragments) {
                t.containerProbeError = errors.New("access forbidden; check RBAC permissions for pod/exec")
                return t.containerProbeError
            } else {
                posixErr = errors.New(message)
            }
        } else {
            posixErr = err
        }
    } else if !utf8.Valid(envBytes) {
        t.containerProbeError = errors.New("non-UTF-8 POSIX environment")
        return t.containerProbeError
    } else if env := string(envBytes); env == "" {
        t.containerProbeError = errors.New("empty POSIX environment")
        return t.containerProbeError
    } else {
        posixEnv = env
        // Use findEnvironmentVariable to correctly parse the block.
        // NOTE: environment.ParseBlock returns []string, not map[string]string.
        if h, ok := findEnvironmentVariable(env, "HOME"); !ok {
            // HOME not found - could be Windows or non-standard POSIX
            posixErr = errors.New("HOME not found in environment")
        } else if h == "" {
            t.containerProbeError = errors.New("empty POSIX home directory")
            return t.containerProbeError
        } else {
            home = h
        }
    }

    // Even if we found HOME, check if this might be a Windows container.
    // Windows containers often have HOME set (e.g., by Git Bash, MSYS, etc.),
    // so we need to also check for USERPROFILE to distinguish.
    if home != "" && posixEnv != "" {
        if userprofile, ok := findEnvironmentVariable(posixEnv, "USERPROFILE"); ok && userprofile != "" {
            // USERPROFILE present alongside HOME suggests Windows.
            // Run Windows-specific probe to confirm.
            if command, err := t.commandDirect("cmd", "/c", "echo %OS%"); err == nil {
                if osBytes, err := command.Output(); err == nil {
                    osStr := strings.TrimSpace(string(osBytes))
                    if strings.Contains(strings.ToLower(osStr), "windows") {
                        // Confirmed Windows - use USERPROFILE instead.
                        home = userprofile
                        windows = true
                    }
                }
            }
        }
    }

    // If we didn't find a POSIX home directory, attempt Windows probing.
    if home == "" {
        if command, err := t.commandDirect("cmd", "/c", "set"); err != nil {
            return fmt.Errorf("unable to set up kubectl invocation: %w", err)
        } else if envBytes, err := command.Output(); err != nil {
            if message := process.ExtractExitErrorMessage(err); message != "" {
                windowsErr = errors.New(message)
            } else {
                windowsErr = err
            }
        } else if !utf8.Valid(envBytes) {
            t.containerProbeError = errors.New("non-UTF-8 Windows environment")
            return t.containerProbeError
        } else if env := string(envBytes); env == "" {
            t.containerProbeError = errors.New("empty Windows environment")
            return t.containerProbeError
        } else {
            // Use findEnvironmentVariable to correctly parse the block.
            if h, ok := findEnvironmentVariable(env, "USERPROFILE"); !ok {
                t.containerProbeError = errors.New("unable to find home directory in Windows environment")
                return t.containerProbeError
            } else if h == "" {
                t.containerProbeError = errors.New("empty Windows home directory")
                return t.containerProbeError
            } else {
                home = h
                windows = true
            }
        }
    }

    // If this is a POSIX container, check if shell is available for workdir support.
    // This is needed because we wrap commands with "sh -c" for working directory.
    if !windows && home != "" {
        if command, err := t.commandDirect("sh", "-c", "echo ok"); err == nil {
            if output, err := command.Output(); err != nil || strings.TrimSpace(string(output)) != "ok" {
                hasShell = false
            }
        } else {
            hasShell = false
        }
    }

    // If both probing mechanisms failed, create a combined error.
    if home == "" {
        t.containerProbeError = fmt.Errorf(
            "container probing failed under POSIX hypothesis (%v) and Windows hypothesis (%v)",
            posixErr,
            windowsErr,
        )
        return t.containerProbeError
    }

    // For POSIX containers, probe username and default group (following Docker's pattern).
    var username, group string
    if !windows {
        // Query username.
        if command, err := t.commandDirect("id", "-un"); err != nil {
            return fmt.Errorf("unable to set up kubectl invocation: %w", err)
        } else if usernameBytes, err := command.Output(); err != nil {
            t.containerProbeError = errors.New("unable to probe POSIX username")
            return t.containerProbeError
        } else if !utf8.Valid(usernameBytes) {
            t.containerProbeError = errors.New("non-UTF-8 POSIX username")
            return t.containerProbeError
        } else if u := strings.TrimSpace(string(usernameBytes)); u == "" {
            t.containerProbeError = errors.New("empty POSIX username")
            return t.containerProbeError
        } else if t.user != "" && u != t.user {
            // If user was specified in URL, validate it matches the container user.
            // Note: kubectl doesn't support running as a different user, so this
            // is a validation check, not enforcement.
            t.containerProbeError = fmt.Errorf("specified user %q does not match container user %q", t.user, u)
            return t.containerProbeError
        } else {
            username = u
        }

        // Query default group name.
        if command, err := t.commandDirect("id", "-gn"); err != nil {
            return fmt.Errorf("unable to set up kubectl invocation: %w", err)
        } else if groupBytes, err := command.Output(); err != nil {
            t.containerProbeError = errors.New("unable to probe POSIX group name")
            return t.containerProbeError
        } else if !utf8.Valid(groupBytes) {
            t.containerProbeError = errors.New("non-UTF-8 POSIX group name")
            return t.containerProbeError
        } else if g := strings.TrimSpace(string(groupBytes)); g == "" {
            t.containerProbeError = errors.New("empty POSIX group name")
            return t.containerProbeError
        } else {
            group = g
        }
    }

    // Store values.
    t.containerIsWindows = windows
    t.containerHomeDirectory = home
    t.containerUser = username
    t.containerUserGroup = group
    t.containerHasShell = hasShell

    return nil
}

// Copy implements the Copy method of agent.Transport.
func (t *kubernetesTransport) Copy(localPath, remoteName string) error {
    // Ensure container has been probed.
    if err := t.probeContainer(); err != nil {
        return fmt.Errorf("unable to probe container: %w", err)
    }

    // Compute the remote path inside the container.
    var remotePath string
    if t.containerIsWindows {
        remotePath = fmt.Sprintf("%s\\%s", t.containerHomeDirectory, remoteName)
    } else {
        remotePath = fmt.Sprintf("%s/%s", t.containerHomeDirectory, remoteName)
    }

    // Build kubectl cp arguments.
    // Format: kubectl cp [options] <src> <namespace>/<pod>:<dest> [-c container]
    // IMPORTANT: Container flag comes AFTER the paths for kubectl cp.
    var kubectlArguments []string

    // Add base connection flags FIRST.
    kubectlArguments = append(kubectlArguments, t.buildBaseFlags()...)

    // Add cp subcommand.
    kubectlArguments = append(kubectlArguments, "cp")

    // Add source (local path).
    kubectlArguments = append(kubectlArguments, localPath)

    // Add destination: pod:path (namespace already in base flags).
    // Note: We use just pod:path since -n already specifies the namespace.
    containerDest := fmt.Sprintf("%s:%s", t.pod, remotePath)
    kubectlArguments = append(kubectlArguments, containerDest)

    // Add container specification AFTER paths (kubectl cp syntax requirement).
    if t.effectiveContainer != "" {
        kubectlArguments = append(kubectlArguments, "-c", t.effectiveContainer)
    }

    // Create the command.
    kubectlCommand, err := kubernetes.Command(context.Background(), kubectlArguments...)
    if err != nil {
        return fmt.Errorf("unable to set up kubectl invocation: %w", err)
    }

    // Set process attributes.
    kubectlCommand.SysProcAttr = transport.ProcessAttributes()

    // Build environment.
    cmdEnv := kubectlCommand.Environ()
    cmdEnv = setKubernetesVariables(cmdEnv, t.environment)
    kubectlCommand.Env = cmdEnv

    // Run the copy operation.
    if output, err := kubectlCommand.CombinedOutput(); err != nil {
        outputStr := strings.TrimSpace(string(output))
        
        // Check for specific error conditions that require clear messages.
        // Classification order matches probeContainer and ClassifyError for consistency:
        // 1. Cluster/context/kubeconfig errors (configuration issues)
        // 2. Namespace errors
        // 3. Pod/container state errors
        // 4. Container-specific errors (tar, read-only)
        // 5. Permission errors
        
        // 1. Check for cluster/context errors first - these are configuration issues.
        if isContextOrClusterError(outputStr) {
            if strings.Contains(strings.ToLower(outputStr), "context") {
                return errors.New("Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG")
            }
            return errors.New("unable to connect to Kubernetes cluster; check cluster connectivity and credentials")
        }
        if containsAnyFragment(outputStr, kubeconfigNotFoundFragments) {
            return errors.New("kubeconfig file not found or invalid; check --kubernetes-kubeconfig or KUBECONFIG environment variable")
        }
        
        // 2. Check for namespace errors.
        if isNamespaceError(outputStr) {
            return fmt.Errorf("namespace %q not found", t.effectiveNamespace)
        }
        
        // 3. Check for pod/container state errors.
        if containsAnyFragment(outputStr, podNotFoundFragments) && !isNamespaceError(outputStr) {
            return errors.New("pod does not exist")
        }
        if containsAnyFragment(outputStr, podNotRunningFragments) {
            return errors.New("pod is not running")
        }
        if containsAnyFragment(outputStr, containerNotFoundFragments) {
            return fmt.Errorf("container %q not found in pod", t.effectiveContainer)
        }
        if containsAnyFragment(outputStr, multiContainerFragments) {
            return errors.New("pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)")
        }
        
        // 4. Check for container-specific errors (tar, read-only).
        if containsAnyFragment(outputStr, tarNotFoundFragments) {
            return errors.New("tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)")
        }
        if containsAnyFragment(outputStr, readOnlyFilesystemFragments) {
            return errors.New("container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)")
        }
        
        // 5. Check for permission errors.
        if containsAnyFragment(outputStr, forbiddenFragments) {
            return errors.New("access forbidden; check RBAC permissions for pod/exec")
        }
        
        // Generic error with output.
        if outputStr != "" {
            return fmt.Errorf("unable to run kubectl cp command: %w (output: %s)", err, outputStr)
        }
        return fmt.Errorf("unable to run kubectl cp command: %w", err)
    }

    // Set ownership for POSIX containers using chown.
    // For Windows, file permissions are inherited from the destination directory.
    // Note: We use commandDirect which doesn't use a shell wrapper, so the path
    // is passed directly to the chown command as an argument (no shell escaping needed).
    if !t.containerIsWindows {
        chownCommand, err := t.commandDirect(
            "chown",
            fmt.Sprintf("%s:%s", t.containerUser, t.containerUserGroup),
            remotePath,
        )
        if err != nil {
            return fmt.Errorf("unable to set up kubectl invocation: %w", err)
        }
        if err := chownCommand.Run(); err != nil {
            return fmt.Errorf("unable to set ownership of copied file: %w", err)
        }
    }

    return nil
}

// Command implements the Command method of agent.Transport.
func (t *kubernetesTransport) Command(cmd string) (*exec.Cmd, error) {
    // Ensure container has been probed.
    if err := t.probeContainer(); err != nil {
        return nil, fmt.Errorf("unable to probe container: %w", err)
    }

    // Create command with home directory as working directory.
    return t.command(cmd, t.containerHomeDirectory)
}

// ClassifyError implements the ClassifyError method of agent.Transport.
func (t *kubernetesTransport) ClassifyError(
    processState *os.ProcessState,
    errorOutput string,
) (bool, bool, error) {
    // Ensure container has been probed.
    if err := t.probeContainer(); err != nil {
        return false, false, fmt.Errorf("unable to probe container: %w", err)
    }

    // Classification order matches probeContainer and Copy for consistency:
    // 1. Cluster/context/kubeconfig errors (configuration issues)
    // 2. Namespace errors
    // 3. Pod/container state errors
    // 4. Container-specific errors (tar, read-only)
    // 5. Permission errors
    // 6. Lifecycle events (pod restart)
    // 7. Command not found (triggers agent install)
    
    // 1. Check for cluster/context errors first - these are configuration issues.
    if isContextOrClusterError(errorOutput) {
        if strings.Contains(strings.ToLower(errorOutput), "context") {
            return false, false, errors.New("Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG")
        }
        return false, false, errors.New("unable to connect to Kubernetes cluster; check cluster connectivity and credentials")
    }
    if containsAnyFragment(errorOutput, kubeconfigNotFoundFragments) {
        return false, false, errors.New("kubeconfig file not found or invalid; check --kubernetes-kubeconfig or KUBECONFIG environment variable")
    }

    // 2. Check for namespace errors.
    if isNamespaceError(errorOutput) {
        return false, false, fmt.Errorf("namespace %q not found", t.effectiveNamespace)
    }

    // 3. Check for pod/container state errors.
    if containsAnyFragment(errorOutput, podNotFoundFragments) && !isNamespaceError(errorOutput) {
        return false, false, errors.New("pod does not exist")
    }
    if containsAnyFragment(errorOutput, podNotRunningFragments) {
        return false, false, errors.New("pod is not running")
    }
    if containsAnyFragment(errorOutput, containerNotFoundFragments) {
        return false, false, fmt.Errorf("container %q not found in pod", t.effectiveContainer)
    }
    if containsAnyFragment(errorOutput, multiContainerFragments) {
        return false, false, errors.New("pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)")
    }

    // 4. Check for container-specific errors (tar, read-only).
    if containsAnyFragment(errorOutput, tarNotFoundFragments) {
        return false, false, errors.New("tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)")
    }
    if containsAnyFragment(errorOutput, readOnlyFilesystemFragments) {
        return false, false, errors.New("container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)")
    }

    // 5. Check for permission errors.
    if containsAnyFragment(errorOutput, forbiddenFragments) {
        return false, false, errors.New("access forbidden; check RBAC permissions for pod/exec")
    }

    // 6. Check for pod restart/deletion (lifecycle issues).
    if containsAnyFragment(errorOutput, podRestartedFragments) {
        // Signal that the agent needs reinstallation due to lifecycle event.
        return true, false, nil
    }

    // 7. Check POSIX shell exit codes for command not found conditions.
    if process.IsPOSIXShellInvalidCommand(processState) {
        return true, false, nil
    }
    if process.IsPOSIXShellCommandNotFound(processState) {
        return true, false, nil
    }

    // Check for POSIX command-not-found patterns in output.
    if process.OutputIsPOSIXCommandNotFound(errorOutput) {
        return true, false, nil
    }

    // Check for Windows command-not-found patterns.
    if process.OutputIsWindowsInvalidCommand(errorOutput) {
        // Windows invalid command - may indicate POSIX command in Windows container.
        return false, true, nil
    }
    if process.OutputIsWindowsCommandNotFound(errorOutput) {
        return true, true, nil
    }

    // Unable to classify the error.
    return false, false, errors.New("unknown process exit error")
}
```

**Environment Helper (`pkg/agent/transport/kubernetes/environment.go`):**

```go
// pkg/agent/transport/kubernetes/environment.go
package kubernetes

import (
    "strings"
)

// kubernetesEnvironmentVariables is the list of environment variables that
// should be passed through to kubectl commands.
var kubernetesEnvironmentVariables = []string{
    "KUBECONFIG",
}

// setKubernetesVariables sets Kubernetes environment variables in the
// environment slice. It filters existing values and adds the captured ones.
// This follows the Docker transport's pattern of replacing rather than
// appending to prevent environment leakage.
func setKubernetesVariables(environment []string, values map[string]string) []string {
    // First, filter out any existing Kubernetes variables to prevent leakage.
    var filtered []string
    for _, env := range environment {
        isKubeVar := false
        for _, kubeVar := range kubernetesEnvironmentVariables {
            if strings.HasPrefix(env, kubeVar+"=") {
                isKubeVar = true
                break
            }
        }
        if !isKubeVar {
            filtered = append(filtered, env)
        }
    }

    // Now add the captured values.
    for _, variable := range kubernetesEnvironmentVariables {
        if value, ok := values[variable]; ok {
            filtered = append(filtered, variable+"="+value)
        }
    }

    return filtered
}
```

---

### Phase 4: Protocol Handlers

#### 4.1 Synchronization Protocol Handler

**File: `pkg/synchronization/protocols/kubernetes/protocol.go`**

```go
package kubernetes

import (
    "context"
    "fmt"
    "io"

    "github.com/mutagen-io/mutagen/pkg/agent"
    "github.com/mutagen-io/mutagen/pkg/agent/transport/kubernetes"
    "github.com/mutagen-io/mutagen/pkg/logging"
    "github.com/mutagen-io/mutagen/pkg/synchronization"
    "github.com/mutagen-io/mutagen/pkg/synchronization/endpoint/remote"
    urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// protocolHandler implements the synchronization.ProtocolHandler interface for
// connecting to remote endpoints inside Kubernetes pods.
type protocolHandler struct{}

// dialResult provides asynchronous agent dialing results.
type dialResult struct {
    stream io.ReadWriteCloser
    error  error
}

// Connect connects to a Kubernetes endpoint.
func (h *protocolHandler) Connect(
    ctx context.Context,
    logger *logging.Logger,
    url *urlpkg.URL,
    prompter string,
    session string,
    version synchronization.Version,
    configuration *synchronization.Configuration,
    alpha bool,
) (synchronization.Endpoint, error) {
    // Verify URL kind and protocol.
    if url.Kind != urlpkg.Kind_Synchronization {
        panic("non-synchronization URL dispatched to synchronization protocol handler")
    } else if url.Protocol != urlpkg.Protocol_Kubernetes {
        panic("non-Kubernetes URL dispatched to Kubernetes protocol handler")
    }

    // Parse the URL host to extract namespace, pod, and container.
    // Use the centralized parser from the url package.
    parsed, err := urlpkg.ParseKubernetesHost(url.Host)
    if err != nil {
        return nil, fmt.Errorf("invalid Kubernetes URL host: %w", err)
    }

    // Create a Kubernetes agent transport.
    transport, err := kubernetes.NewTransport(
        parsed.Pod,
        parsed.Namespace,
        parsed.Container,
        url.User,
        url.Environment,
        url.Parameters,
        prompter,
    )
    if err != nil {
        return nil, fmt.Errorf("unable to create Kubernetes transport: %w", err)
    }

    // Create a channel to deliver the dialing result.
    results := make(chan dialResult)

    // Perform dialing in a background Goroutine.
    go func() {
        stream, err := agent.Dial(logger, transport, agent.CommandSynchronizer, prompter)
        select {
        case results <- dialResult{stream, err}:
        case <-ctx.Done():
            if stream != nil {
                stream.Close()
            }
        }
    }()

    // Wait for dialing results or cancellation.
    var stream io.ReadWriteCloser
    select {
    case result := <-results:
        if result.error != nil {
            return nil, fmt.Errorf("unable to dial agent endpoint: %w", result.error)
        }
        stream = result.stream
    case <-ctx.Done():
        return nil, context.Canceled
    }

    // Create the endpoint client.
    return remote.NewEndpoint(logger, stream, url.Path, session, version, configuration, alpha)
}

func init() {
    // Register the Kubernetes protocol handler.
    synchronization.ProtocolHandlers[urlpkg.Protocol_Kubernetes] = &protocolHandler{}
}
```

#### 4.2 Forwarding Protocol Handler

**File: `pkg/forwarding/protocols/kubernetes/protocol.go`**

```go
package kubernetes

import (
    "context"
    "fmt"
    "io"

    "github.com/mutagen-io/mutagen/pkg/agent"
    "github.com/mutagen-io/mutagen/pkg/agent/transport/kubernetes"
    "github.com/mutagen-io/mutagen/pkg/forwarding"
    "github.com/mutagen-io/mutagen/pkg/forwarding/endpoint/remote"
    "github.com/mutagen-io/mutagen/pkg/logging"
    urlpkg "github.com/mutagen-io/mutagen/pkg/url"
    forwardingurlpkg "github.com/mutagen-io/mutagen/pkg/url/forwarding"
)

// protocolHandler implements the forwarding.ProtocolHandler interface.
type protocolHandler struct{}

// dialResult provides asynchronous agent dialing results.
type dialResult struct {
    stream io.ReadWriteCloser
    error  error
}

// Connect connects to a Kubernetes forwarding endpoint.
func (p *protocolHandler) Connect(
    ctx context.Context,
    logger *logging.Logger,
    url *urlpkg.URL,
    prompter string,
    session string,
    version forwarding.Version,
    configuration *forwarding.Configuration,
    source bool,
) (forwarding.Endpoint, error) {
    // Verify URL kind and protocol.
    if url.Kind != urlpkg.Kind_Forwarding {
        panic("non-forwarding URL dispatched to forwarding protocol handler")
    } else if url.Protocol != urlpkg.Protocol_Kubernetes {
        panic("non-Kubernetes URL dispatched to Kubernetes protocol handler")
    }

    // Parse the target specification.
    protocol, address, err := forwardingurlpkg.Parse(url.Path)
    if err != nil {
        return nil, fmt.Errorf("unable to parse target specification: %w", err)
    }

    // Parse the URL host to extract namespace, pod, and container.
    // Use the centralized parser from the url package.
    parsed, err := urlpkg.ParseKubernetesHost(url.Host)
    if err != nil {
        return nil, fmt.Errorf("invalid Kubernetes URL host: %w", err)
    }

    // Create a Kubernetes agent transport.
    transport, err := kubernetes.NewTransport(
        parsed.Pod,
        parsed.Namespace,
        parsed.Container,
        url.User,
        url.Environment,
        url.Parameters,
        prompter,
    )
    if err != nil {
        return nil, fmt.Errorf("unable to create Kubernetes transport: %w", err)
    }

    // Create a channel to deliver the dialing result.
    results := make(chan dialResult)

    // Perform dialing in a background Goroutine.
    go func() {
        stream, err := agent.Dial(logger, transport, agent.CommandForwarder, prompter)
        select {
        case results <- dialResult{stream, err}:
        case <-ctx.Done():
            if stream != nil {
                stream.Close()
            }
        }
    }()

    // Wait for dialing results or cancellation.
    var stream io.ReadWriteCloser
    select {
    case result := <-results:
        if result.error != nil {
            return nil, fmt.Errorf("unable to dial agent endpoint: %w", result.error)
        }
        stream = result.stream
    case <-ctx.Done():
        return nil, context.Canceled
    }

    // Create the endpoint.
    return remote.NewEndpoint(logger, stream, version, configuration, protocol, address, source)
}

func init() {
    // Register the Kubernetes protocol handler.
    forwarding.ProtocolHandlers[urlpkg.Protocol_Kubernetes] = &protocolHandler{}
}
```

---

### Phase 5: CLI Integration

#### 5.1 Command-Line Flags (`cmd/external/kubernetes.go`)

```go
// cmd/external/kubernetes.go
package external

import (
    "github.com/spf13/pflag"

    "github.com/mutagen-io/mutagen/pkg/kubernetes"
    urlpkg "github.com/mutagen-io/mutagen/pkg/url"
)

// KubernetesFlags stores Kubernetes command line flags.
var KubernetesFlags struct {
    // Kubeconfig is the path to the kubeconfig file.
    Kubeconfig string
    // Context is the Kubernetes context to use.
    Context string
    // Namespace is the Kubernetes namespace to use.
    Namespace string
    // Container is the container name within the pod.
    Container string
}

// RegisterKubernetesFlags registers Kubernetes flags with a flag set.
func RegisterKubernetesFlags(flags *pflag.FlagSet) {
    flags.StringVar(&KubernetesFlags.Kubeconfig, "kubernetes-kubeconfig", "",
        "Specify the path to the kubeconfig file")
    flags.StringVar(&KubernetesFlags.Context, "kubernetes-context", "",
        "Specify the Kubernetes context to use")
    flags.StringVar(&KubernetesFlags.Namespace, "kubernetes-namespace", "",
        "Specify the Kubernetes namespace (overrides URL namespace)")
    flags.StringVar(&KubernetesFlags.Container, "kubernetes-container", "",
        "Specify the container name within the pod (overrides URL container)")
}

// KubernetesConnectionFlags returns connection flags from CLI flags.
func KubernetesConnectionFlags() *kubernetes.ConnectionFlags {
    return &kubernetes.ConnectionFlags{
        Kubeconfig: KubernetesFlags.Kubeconfig,
        Context:    KubernetesFlags.Context,
        Namespace:  KubernetesFlags.Namespace,
        Container:  KubernetesFlags.Container,
    }
}

// ApplyKubernetesParametersToURL applies CLI flags to a Kubernetes URL's
// parameters. CLI flags take precedence over URL components.
// This function should be called after URL parsing but before session creation.
// It safely handles nil URLs and non-Kubernetes protocols.
func ApplyKubernetesParametersToURL(url *urlpkg.URL) {
    // Guard against nil URL.
    if url == nil {
        return
    }

    // Only process Kubernetes URLs.
    if url.Protocol != urlpkg.Protocol_Kubernetes {
        return
    }

    // Initialize parameters map if nil.
    if url.Parameters == nil {
        url.Parameters = make(map[string]string)
    }

    // Apply CLI flags to parameters. These will override URL components
    // when the transport is created.
    flags := KubernetesConnectionFlags()
    if flags == nil {
        return
    }
    params := flags.ToURLParameters()
    for k, v := range params {
        if v != "" {
            url.Parameters[k] = v
        }
    }
}
```

#### 5.2 Sync/Forward Command Updates

**Modifications to `cmd/mutagen/sync/create.go`:**

```go
import (
    // ... existing imports ...
    "github.com/mutagen-io/mutagen/cmd/external"
)

func createMain(_ *cobra.Command, arguments []string) error {
    // ... existing URL parsing ...
    alpha, err := url.Parse(arguments[0], url.Kind_Synchronization, true)
    if err != nil {
        return fmt.Errorf("unable to parse alpha URL: %w", err)
    }
    beta, err := url.Parse(arguments[1], url.Kind_Synchronization, false)
    if err != nil {
        return fmt.Errorf("unable to parse beta URL: %w", err)
    }

    // Apply Kubernetes CLI flags to URLs if applicable.
    // CLI flags override URL components for consistency.
    external.ApplyKubernetesParametersToURL(alpha)
    external.ApplyKubernetesParametersToURL(beta)

    // ... rest of the function ...
}

func init() {
    // ... existing flag registration ...
    
    // Register Kubernetes flags.
    external.RegisterKubernetesFlags(createCommand.Flags())
}
```

**Same pattern for `cmd/mutagen/forward/create.go`.**

#### 5.3 URL/Flag Conflict Resolution Strategy

The following precedence rules apply:

1. **CLI flags** (`--kubernetes-namespace`, `--kubernetes-container`) take highest precedence
2. **URL parameters** (from previous session creation) are second
3. **URL host components** (`namespace/pod/container`) are the base values

This allows users to:
- Specify the full target in the URL: `kubernetes://prod/my-pod/main:/data`
- Override specific components: `mutagen sync create --kubernetes-container=sidecar kubernetes://prod/my-pod:/data`

**Conflict detection:** If both URL and CLI specify the same component with different values, the CLI wins silently. This matches the behavior of other transports (SSH, Docker).

**Design Decision (Intentional):** Silent precedence was chosen over warning/error on conflicts because:
1. It matches existing Mutagen transport behavior for consistency
2. CLI flags are often used for one-time overrides (e.g., testing against a different namespace)
3. URL parameters are persisted in session state, so the resolved value is always visible via `mutagen sync list`
4. Warning on every invocation would be noisy for legitimate override use cases

If future UX research suggests warnings are preferred, they can be added behind a `--warn-on-override` flag.

---

### Phase 6: Testing

#### 6.1 Unit Tests

**URL Parsing Tests (`pkg/url/parse_kubernetes_test.go`):**

```go
package url

import (
    "reflect"
    "testing"
)

func TestParseKubernetesHost(t *testing.T) {
    testCases := []struct {
        name        string
        host        string
        expected    *ParsedKubernetesHost
        expectError bool
        errorMsg    string
    }{
        {
            name: "namespace and pod",
            host: "default/my-pod",
            expected: &ParsedKubernetesHost{
                Namespace: "default",
                Pod:       "my-pod",
            },
        },
        {
            name: "namespace, pod, and container",
            host: "prod/my-pod/main",
            expected: &ParsedKubernetesHost{
                Namespace: "prod",
                Pod:       "my-pod",
                Container: "main",
            },
        },
        {
            name:        "pod only - error",
            host:        "my-pod",
            expectError: true,
            errorMsg:    "namespace is required",
        },
        {
            name:        "empty namespace - error",
            host:        "/my-pod",
            expectError: true,
            errorMsg:    "empty namespace",
        },
        {
            name:        "empty pod - error",
            host:        "default/",
            expectError: true,
            errorMsg:    "empty pod name",
        },
        {
            name:        "empty container - error",
            host:        "default/my-pod/",
            expectError: true,
            errorMsg:    "empty container name",
        },
        {
            name:        "too many components - error",
            host:        "context/namespace/pod/container",
            expectError: true,
            errorMsg:    "invalid host format",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := ParseKubernetesHost(tc.host)
            if tc.expectError {
                if err == nil {
                    t.Errorf("expected error containing %q, got nil", tc.errorMsg)
                } else if !strings.Contains(err.Error(), tc.errorMsg) {
                    t.Errorf("expected error containing %q, got %q", tc.errorMsg, err.Error())
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if !reflect.DeepEqual(result, tc.expected) {
                t.Errorf("expected %+v, got %+v", tc.expected, result)
            }
        })
    }
}

func TestParseKubernetesURL(t *testing.T) {
    testCases := []struct {
        name        string
        raw         string
        kind        Kind
        first       bool
        expectError bool
        expected    *URL
    }{
        {
            name:  "basic sync URL",
            raw:   "kubernetes://default/my-pod:/app/data",
            kind:  Kind_Synchronization,
            first: true,
            expected: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                Host:     "default/my-pod",
                Path:     "/app/data",
            },
        },
        {
            name:  "sync URL with container",
            raw:   "kubernetes://prod/my-pod/main:/data",
            kind:  Kind_Synchronization,
            first: true,
            expected: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                Host:     "prod/my-pod/main",
                Path:     "/data",
            },
        },
        {
            name:  "sync URL with user",
            raw:   "kubernetes://user@default/my-pod:/home",
            kind:  Kind_Synchronization,
            first: true,
            expected: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                User:     "user",
                Host:     "default/my-pod",
                Path:     "/home",
            },
        },
        {
            name:  "sync URL with home-relative path",
            raw:   "kubernetes://default/my-pod:~/data",
            kind:  Kind_Synchronization,
            first: true,
            expected: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                Host:     "default/my-pod",
                Path:     "~/data",
            },
        },
        {
            name:  "forwarding URL",
            raw:   "kubernetes://default/my-pod:tcp:localhost:8080",
            kind:  Kind_Forwarding,
            first: true,
            expected: &URL{
                Kind:     Kind_Forwarding,
                Protocol: Protocol_Kubernetes,
                Host:     "default/my-pod",
                Path:     "tcp:localhost:8080",
            },
        },
        {
            name:        "missing path separator",
            raw:         "kubernetes://default/my-pod",
            kind:        Kind_Synchronization,
            first:       true,
            expectError: true,
        },
        {
            name:        "invalid host - no namespace",
            raw:         "kubernetes://my-pod:/data",
            kind:        Kind_Synchronization,
            first:       true,
            expectError: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := Parse(tc.raw, tc.kind, tc.first)
            if tc.expectError {
                if err == nil {
                    t.Error("expected error but got none")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            // Compare relevant fields (Environment is populated from actual env).
            if result.Kind != tc.expected.Kind {
                t.Errorf("Kind: expected %v, got %v", tc.expected.Kind, result.Kind)
            }
            if result.Protocol != tc.expected.Protocol {
                t.Errorf("Protocol: expected %v, got %v", tc.expected.Protocol, result.Protocol)
            }
            if result.User != tc.expected.User {
                t.Errorf("User: expected %v, got %v", tc.expected.User, result.User)
            }
            if result.Host != tc.expected.Host {
                t.Errorf("Host: expected %v, got %v", tc.expected.Host, result.Host)
            }
            if result.Path != tc.expected.Path {
                t.Errorf("Path: expected %v, got %v", tc.expected.Path, result.Path)
            }
        })
    }
}

func TestKubernetesURLFormatRoundTrip(t *testing.T) {
    testCases := []struct {
        name string
        url  *URL
    }{
        {
            name: "basic sync",
            url: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                Host:     "default/my-pod",
                Path:     "/app/data",
            },
        },
        {
            name: "with container",
            url: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                Host:     "prod/my-pod/main",
                Path:     "/data",
            },
        },
        {
            name: "with user",
            url: &URL{
                Kind:     Kind_Synchronization,
                Protocol: Protocol_Kubernetes,
                User:     "appuser",
                Host:     "default/my-pod",
                Path:     "/home/appuser",
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Format the URL.
            formatted := Format(tc.url)

            // Parse it back.
            parsed, err := Parse(formatted, tc.url.Kind, true)
            if err != nil {
                t.Fatalf("round-trip parse failed: %v", err)
            }

            // Compare.
            if parsed.Protocol != tc.url.Protocol {
                t.Errorf("Protocol mismatch: %v != %v", parsed.Protocol, tc.url.Protocol)
            }
            if parsed.User != tc.url.User {
                t.Errorf("User mismatch: %v != %v", parsed.User, tc.url.User)
            }
            if parsed.Host != tc.url.Host {
                t.Errorf("Host mismatch: %v != %v", parsed.Host, tc.url.Host)
            }
            if parsed.Path != tc.url.Path {
                t.Errorf("Path mismatch: %v != %v", parsed.Path, tc.url.Path)
            }
        })
    }
}
```

**Environment Handling Tests (`pkg/agent/transport/kubernetes/environment_test.go`):**

```go
package kubernetes

import (
    "reflect"
    "testing"
)

func TestSetKubernetesVariables(t *testing.T) {
    testCases := []struct {
        name        string
        environment []string
        values      map[string]string
        expected    []string
    }{
        {
            name:        "add KUBECONFIG",
            environment: []string{"PATH=/usr/bin", "HOME=/home/user"},
            values:      map[string]string{"KUBECONFIG": "/home/user/.kube/config"},
            expected:    []string{"PATH=/usr/bin", "HOME=/home/user", "KUBECONFIG=/home/user/.kube/config"},
        },
        {
            name:        "replace existing KUBECONFIG",
            environment: []string{"PATH=/usr/bin", "KUBECONFIG=/old/path"},
            values:      map[string]string{"KUBECONFIG": "/new/path"},
            expected:    []string{"PATH=/usr/bin", "KUBECONFIG=/new/path"},
        },
        {
            name:        "no values to set",
            environment: []string{"PATH=/usr/bin"},
            values:      map[string]string{},
            expected:    []string{"PATH=/usr/bin"},
        },
        {
            name:        "value not in captured set - ignored",
            environment: []string{"PATH=/usr/bin"},
            values:      map[string]string{"SOME_OTHER_VAR": "value"},
            expected:    []string{"PATH=/usr/bin"},
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := setKubernetesVariables(tc.environment, tc.values)
            if !reflect.DeepEqual(result, tc.expected) {
                t.Errorf("expected %v, got %v", tc.expected, result)
            }
        })
    }
}

func TestFindEnvironmentVariable(t *testing.T) {
    testCases := []struct {
        name     string
        block    string
        variable string
        want     string
        wantOK   bool
    }{
        {
            name:     "simple POSIX",
            block:    "HOME=/home/user\nPATH=/usr/bin\nUSER=testuser",
            variable: "HOME",
            want:     "/home/user",
            wantOK:   true,
        },
        {
            name:     "Windows CRLF",
            block:    "USERPROFILE=C:\\Users\\test\r\nPATH=C:\\Windows\r\n",
            variable: "USERPROFILE",
            want:     "C:\\Users\\test",
            wantOK:   true,
        },
        {
            name:     "not found",
            block:    "HOME=/home/user\nPATH=/usr/bin",
            variable: "NOTFOUND",
            want:     "",
            wantOK:   false,
        },
        {
            name:     "empty value",
            block:    "HOME=\nPATH=/usr/bin",
            variable: "HOME",
            want:     "",
            wantOK:   true,
        },
        {
            name:     "value with equals sign",
            block:    "CONFIG=key=value\nPATH=/usr/bin",
            variable: "CONFIG",
            want:     "key=value",
            wantOK:   true,
        },
        {
            name:     "partial match ignored",
            block:    "HOMEDIR=/data\nHOME=/home/user",
            variable: "HOME",
            want:     "/home/user",
            wantOK:   true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            got, ok := findEnvironmentVariable(tc.block, tc.variable)
            if ok != tc.wantOK {
                t.Errorf("findEnvironmentVariable(%q, %q) ok = %v, want %v",
                    tc.block, tc.variable, ok, tc.wantOK)
            }
            if got != tc.want {
                t.Errorf("findEnvironmentVariable(%q, %q) = %q, want %q",
                    tc.block, tc.variable, got, tc.want)
            }
        })
    }
}
```

**Connection Flags Tests (`pkg/kubernetes/flags_test.go`):**

```go
package kubernetes

import (
    "reflect"
    "testing"
)

func TestLoadConnectionFlagsFromURLParameters(t *testing.T) {
    testCases := []struct {
        name        string
        parameters  map[string]string
        expected    *ConnectionFlags
        expectError bool
    }{
        {
            name:       "empty parameters",
            parameters: map[string]string{},
            expected:   &ConnectionFlags{},
        },
        {
            name: "all parameters",
            parameters: map[string]string{
                "kubeconfig": "/path/to/config",
                "context":    "my-context",
                "namespace":  "my-namespace",
                "container":  "my-container",
            },
            expected: &ConnectionFlags{
                Kubeconfig: "/path/to/config",
                Context:    "my-context",
                Namespace:  "my-namespace",
                Container:  "my-container",
            },
        },
        {
            name:        "empty kubeconfig value",
            parameters:  map[string]string{"kubeconfig": ""},
            expectError: true,
        },
        {
            name:        "unknown parameter",
            parameters:  map[string]string{"unknown": "value"},
            expectError: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := LoadConnectionFlagsFromURLParameters(tc.parameters)
            if tc.expectError {
                if err == nil {
                    t.Error("expected error but got none")
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if !reflect.DeepEqual(result, tc.expected) {
                t.Errorf("expected %+v, got %+v", tc.expected, result)
            }
        })
    }
}

func TestConnectionFlagsRoundTrip(t *testing.T) {
    original := &ConnectionFlags{
        Kubeconfig: "/path/to/config",
        Context:    "my-context",
        Namespace:  "my-namespace",
        Container:  "my-container",
    }

    params := original.ToURLParameters()
    loaded, err := LoadConnectionFlagsFromURLParameters(params)
    if err != nil {
        t.Fatalf("round-trip failed: %v", err)
    }

    if !reflect.DeepEqual(original, loaded) {
        t.Errorf("round-trip mismatch: %+v != %+v", original, loaded)
    }
}

func TestToFlags(t *testing.T) {
    flags := &ConnectionFlags{
        Kubeconfig: "/path/to/config",
        Context:    "my-context",
        Namespace:  "my-namespace",
        Container:  "my-container", // Should NOT be in output.
    }

    result := flags.ToFlags()
    expected := []string{
        "--kubeconfig", "/path/to/config",
        "--context", "my-context",
        "-n", "my-namespace",
    }

    if !reflect.DeepEqual(result, expected) {
        t.Errorf("expected %v, got %v", expected, result)
    }

    // Verify container is NOT included (it's command-specific).
    for _, f := range result {
        if f == "-c" || f == "my-container" {
            t.Error("container flag should not be in base flags")
        }
    }
}
```

**Transport Command Construction Tests (`pkg/agent/transport/kubernetes/transport_test.go`):**

```go
package kubernetes

import (
    "strings"
    "testing"
)

func TestBuildBaseFlags(t *testing.T) {
    transport := &kubernetesTransport{
        kubeconfig:         "/path/to/config",
        kubeContext:        "my-context",
        effectiveNamespace: "my-namespace",
    }

    flags := transport.buildBaseFlags()

    // Verify order and content.
    expected := []string{
        "--kubeconfig", "/path/to/config",
        "--context", "my-context",
        "-n", "my-namespace",
    }

    if len(flags) != len(expected) {
        t.Fatalf("expected %d flags, got %d", len(expected), len(flags))
    }

    for i, f := range expected {
        if flags[i] != f {
            t.Errorf("flag %d: expected %q, got %q", i, f, flags[i])
        }
    }
}

func TestNewTransportValidation(t *testing.T) {
    testCases := []struct {
        name        string
        pod         string
        namespace   string
        parameters  map[string]string
        expectError bool
        errorMsg    string
    }{
        {
            name:      "valid basic",
            pod:       "my-pod",
            namespace: "default",
        },
        {
            name:        "empty pod",
            pod:         "",
            namespace:   "default",
            expectError: true,
            errorMsg:    "pod name is required",
        },
        {
            name:        "empty namespace",
            pod:         "my-pod",
            namespace:   "",
            expectError: true,
            errorMsg:    "namespace is required",
        },
        {
            name:       "namespace from parameters",
            pod:        "my-pod",
            namespace:  "", // Empty in URL.
            parameters: map[string]string{"namespace": "from-params"},
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            _, err := NewTransport(
                tc.pod, tc.namespace, "", "",
                nil, tc.parameters, "",
            )
            if tc.expectError {
                if err == nil {
                    t.Error("expected error but got none")
                } else if !strings.Contains(err.Error(), tc.errorMsg) {
                    t.Errorf("expected error containing %q, got %q", tc.errorMsg, err.Error())
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
        })
    }
}

func TestShellEscape(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "simple path",
            input:    "/app/data",
            expected: "/app/data",
        },
        {
            name:     "path with spaces",
            input:    "/app/my data",
            expected: "'/app/my data'",
        },
        {
            name:     "path with single quote",
            input:    "/app/it's",
            expected: "'/app/it'\\''s'",
        },
        {
            name:     "path with special chars",
            input:    "/app/$data",
            expected: "'/app/$data'",
        },
        {
            name:     "path with backticks",
            input:    "/app/`whoami`",
            expected: "'/app/`whoami`'",
        },
        {
            name:     "path with glob chars",
            input:    "/app/*.txt",
            expected: "'/app/*.txt'",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := shellEscape(tc.input)
            if result != tc.expected {
                t.Errorf("shellEscape(%q) = %q, want %q", tc.input, result, tc.expected)
            }
        })
    }
}

func TestWindowsEscape(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "simple path",
            input:    "C:\\app\\data",
            expected: "C:\\app\\data",
        },
        {
            name:     "path with spaces",
            input:    "C:\\app\\my data",
            expected: "\"C:\\app\\my data\"",
        },
        {
            name:     "path with pipe",
            input:    "C:\\app|data",
            expected: "\"C:\\app|data\"",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := windowsEscape(tc.input)
            if result != tc.expected {
                t.Errorf("windowsEscape(%q) = %q, want %q", tc.input, result, tc.expected)
            }
        })
    }
}

func TestCommandWithEscaping(t *testing.T) {
    // Test that command() properly escapes paths in shell commands.
    transport := &kubernetesTransport{
        pod:                     "test-pod",
        effectiveNamespace:      "default",
        containerHasShell:       true,
        containerHomeDirectory:  "/home/user",
    }

    // Test with a path containing spaces.
    cmd, err := transport.command("./agent", "/home/user/my data")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    args := cmd.Args
    // Find the shell command argument (after -c).
    for i, arg := range args {
        if arg == "-c" && i+1 < len(args) {
            shellCmd := args[i+1]
            if !strings.Contains(shellCmd, "'/home/user/my data'") {
                t.Errorf("expected escaped path in shell command, got: %s", shellCmd)
            }
            break
        }
    }
}

// classifyCopyError is an exact replica of the error classification logic in Copy.
// This allows unit testing the classification without needing a real kubectl.
// IMPORTANT: Keep this in sync with the Copy method's error classification.
func classifyCopyError(outputStr string, effectiveNamespace, effectiveContainer string) error {
    // Classification order matches Copy method exactly:
    // 1. Cluster/context/kubeconfig errors (configuration issues)
    // 2. Namespace errors
    // 3. Pod/container state errors
    // 4. Container-specific errors (tar, read-only)
    // 5. Permission errors
    
    // 1. Check for cluster/context errors first.
    if isContextOrClusterError(outputStr) {
        if strings.Contains(strings.ToLower(outputStr), "context") {
            return errors.New("Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG")
        }
        return errors.New("unable to connect to Kubernetes cluster; check cluster connectivity and credentials")
    }
    if containsAnyFragment(outputStr, kubeconfigNotFoundFragments) {
        return errors.New("kubeconfig file not found or invalid; check --kubernetes-kubeconfig or KUBECONFIG environment variable")
    }
    
    // 2. Check for namespace errors.
    if isNamespaceError(outputStr) {
        return fmt.Errorf("namespace %q not found", effectiveNamespace)
    }
    
    // 3. Check for pod/container state errors.
    if containsAnyFragment(outputStr, podNotFoundFragments) && !isNamespaceError(outputStr) {
        return errors.New("pod does not exist")
    }
    if containsAnyFragment(outputStr, podNotRunningFragments) {
        return errors.New("pod is not running")
    }
    if containsAnyFragment(outputStr, containerNotFoundFragments) {
        return fmt.Errorf("container %q not found in pod", effectiveContainer)
    }
    if containsAnyFragment(outputStr, multiContainerFragments) {
        return errors.New("pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)")
    }
    
    // 4. Check for container-specific errors (tar, read-only).
    if containsAnyFragment(outputStr, tarNotFoundFragments) {
        return errors.New("tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)")
    }
    if containsAnyFragment(outputStr, readOnlyFilesystemFragments) {
        return errors.New("container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)")
    }
    
    // 5. Check for permission errors.
    if containsAnyFragment(outputStr, forbiddenFragments) {
        return errors.New("access forbidden; check RBAC permissions for pod/exec")
    }
    
    return errors.New("unknown error")
}

func TestProbeErrorClassification(t *testing.T) {
    // Test that error classification produces correct user-facing messages
    // with proper precedence (context > namespace > pod > container > tar/RO).
    
    testCases := []struct {
        name           string
        errorMessage   string
        expectedError  string
        namespace      string
        container      string
    }{
        {
            name:          "kubeconfig not found",
            errorMessage:  "error: unable to load kubeconfig: no such file or directory",
            expectedError: "kubeconfig file not found or invalid; check --kubernetes-kubeconfig or KUBECONFIG environment variable",
            namespace:     "default",
        },
        {
            name:          "context not found",
            errorMessage:  "error: context \"nonexistent\" does not exist",
            expectedError: "Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG",
            namespace:     "default",
        },
        {
            name:          "current-context is not set",
            errorMessage:  "error: current-context is not set",
            expectedError: "Kubernetes context not found or not set; check --kubernetes-context or KUBECONFIG",
            namespace:     "default",
        },
        {
            name:          "cluster connection timeout",
            errorMessage:  "Unable to connect to the server: dial tcp 10.0.0.1:6443: i/o timeout",
            expectedError: "unable to connect to Kubernetes cluster; check cluster connectivity and credentials",
            namespace:     "default",
        },
        {
            name:          "cluster connection refused",
            errorMessage:  "Unable to connect to the server: dial tcp 127.0.0.1:6443: connection refused",
            expectedError: "unable to connect to Kubernetes cluster; check cluster connectivity and credentials",
            namespace:     "default",
        },
        {
            name:          "namespace not found",
            errorMessage:  `Error from server (NotFound): namespaces "nonexistent-ns" not found`,
            expectedError: `namespace "test-ns" not found`,
            namespace:     "test-ns",
        },
        {
            name:          "pod not found",
            errorMessage:  `Error from server (NotFound): pods "nonexistent-pod" not found`,
            expectedError: "pod does not exist",
            namespace:     "default",
        },
        {
            name:          "pod not running",
            errorMessage:  `error: unable to upgrade connection: container not running`,
            expectedError: "pod is not running",
            namespace:     "default",
        },
        {
            name:          "container not found",
            errorMessage:  `error: container "sidecar" not found in pod "my-pod"`,
            expectedError: `container "sidecar" not found in pod`,
            namespace:     "default",
            container:     "sidecar",
        },
        {
            name:          "multi-container pod",
            errorMessage:  `error: you must specify a container`,
            expectedError: "pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)",
            namespace:     "default",
        },
        {
            name:          "forbidden",
            errorMessage:  `Error from server (Forbidden): pods "my-pod" is forbidden`,
            expectedError: "access forbidden; check RBAC permissions for pod/exec",
            namespace:     "default",
        },
        {
            name:          "tar not found",
            errorMessage:  "tar: not found",
            expectedError: "tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)",
            namespace:     "default",
        },
        {
            name:          "read-only filesystem",
            errorMessage:  "tar: /root/.mutagen: Cannot open: Read-only file system",
            expectedError: "container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)",
            namespace:     "default",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := classifyCopyError(tc.errorMessage, tc.namespace, tc.container)
            if err == nil {
                t.Fatal("expected error but got nil")
            }
            // Assert exact match of full user-facing error message.
            if err.Error() != tc.expectedError {
                t.Errorf("expected exact error %q, got %q", tc.expectedError, err.Error())
            }
        })
    }
}

func TestErrorClassificationPrecedence(t *testing.T) {
    // Test that classification precedence is correct:
    // context/cluster > namespace > pod > container > tar/RO > forbidden
    
    // An error containing both context and pod errors should report context.
    mixedContextPod := "error: context not found; also pod not found"
    err := classifyCopyError(mixedContextPod, "ns", "")
    if !strings.Contains(err.Error(), "context") {
        t.Errorf("context should take precedence over pod: %v", err)
    }
    
    // An error containing both namespace and container errors should report namespace.
    mixedNsContainer := `namespaces "bad" not found; container "c" not found`
    err = classifyCopyError(mixedNsContainer, "bad", "c")
    if !strings.Contains(err.Error(), "namespace") {
        t.Errorf("namespace should take precedence over container: %v", err)
    }
    
    // An error with pod state and tar should report pod (state issues before container issues).
    mixedPodTar := "container not running; tar: not found"
    err = classifyCopyError(mixedPodTar, "ns", "")
    if !strings.Contains(err.Error(), "pod is not running") {
        t.Errorf("pod state should take precedence over tar: %v", err)
    }
    
    // An error with tar and forbidden should report tar (before forbidden).
    mixedTarForbidden := "tar: not found; forbidden"
    err = classifyCopyError(mixedTarForbidden, "ns", "")
    if !strings.Contains(err.Error(), "tar command not found") {
        t.Errorf("tar should take precedence over forbidden: %v", err)
    }
}

func TestCopyErrorClassification(t *testing.T) {
    // Test that Copy method error classification produces correct user-facing messages.
    // Uses classifyCopyError helper to test the actual classification logic.
    
    testCases := []struct {
        name          string
        errorOutput   string
        expectedError string
        namespace     string
        container     string
    }{
        {
            name:          "tar not found in container",
            errorOutput:   "tar: not found\ncommand terminated with exit code 127",
            expectedError: "tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)",
            namespace:     "default",
        },
        {
            name:          "tar executable not found",
            errorOutput:   "OCI runtime exec failed: exec failed: unable to start container process: exec: \"tar\": executable file not found in $PATH",
            expectedError: "tar command not found in container; required for agent installation (consider using an image with tar, e.g., alpine, debian, or ubuntu)",
            namespace:     "default",
        },
        {
            name:          "read-only filesystem",
            errorOutput:   "tar: /root/.mutagen: Cannot open: Read-only file system",
            expectedError: "container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)",
            namespace:     "default",
        },
        {
            name:          "EROFS error",
            errorOutput:   "tar: can't create directory '/root/.mutagen': EROFS",
            expectedError: "container filesystem is read-only; cannot install agent (ensure the container has a writable home directory)",
            namespace:     "default",
        },
        {
            name:          "container not found during cp",
            errorOutput:   `error: container "bad" not found in pod`,
            expectedError: `container "bad" not found in pod`,
            namespace:     "default",
            container:     "bad",
        },
        {
            name:          "pod not running during cp",
            errorOutput:   `error: unable to upgrade connection: container not running`,
            expectedError: "pod is not running",
            namespace:     "default",
        },
        {
            name:          "multi-container during cp",
            errorOutput:   `error: you must specify a container in the pod`,
            expectedError: "pod has multiple containers; specify container with --kubernetes-container flag or in URL (namespace/pod/container)",
            namespace:     "default",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            err := classifyCopyError(tc.errorOutput, tc.namespace, tc.container)
            if err == nil {
                t.Fatal("expected error but got nil")
            }
            // Assert exact match of full user-facing error message.
            if err.Error() != tc.expectedError {
                t.Errorf("expected exact error %q, got %q", tc.expectedError, err.Error())
            }
        })
    }
}

func TestClassifyErrorExtended(t *testing.T) {
    transport := &kubernetesTransport{
        pod:                "test-pod",
        effectiveNamespace: "test-ns",
        containerProbed:    true,
    }

    testCases := []struct {
        name          string
        errorOutput   string
        expectInstall bool
        expectError   bool
        errorContains string
    }{
        {
            name:          "context not found",
            errorOutput:   "error: context \"prod\" does not exist",
            expectError:   true,
            errorContains: "context not found",
        },
        {
            name:          "kubeconfig not found",
            errorOutput:   "error: unable to load kubeconfig: no such file or directory",
            expectError:   true,
            errorContains: "kubeconfig",
        },
        {
            name:          "cluster not reachable",
            errorOutput:   "error: dial tcp 10.0.0.1:6443: i/o timeout",
            expectError:   true,
            errorContains: "unable to connect",
        },
        {
            name:          "namespace not found",
            errorOutput:   `Error from server (NotFound): namespaces "bad-ns" not found`,
            expectError:   true,
            errorContains: "namespace",
        },
        {
            name:          "tar not found",
            errorOutput:   "tar: not found",
            expectError:   true,
            errorContains: "tar command not found",
        },
        {
            name:          "read-only filesystem",
            errorOutput:   "cp: cannot create regular file: Read-only file system",
            expectError:   true,
            errorContains: "read-only",
        },
        {
            name:          "pod restarted triggers reinstall",
            errorOutput:   "container has been restarted, you may need to reconnect",
            expectInstall: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            install, _, err := transport.ClassifyError(nil, tc.errorOutput)
            if tc.expectInstall && !install {
                t.Error("expected install=true")
            }
            if tc.expectError {
                if err == nil {
                    t.Error("expected error but got none")
                } else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.errorContains)) {
                    t.Errorf("expected error containing %q, got %q", tc.errorContains, err.Error())
                }
            }
        })
    }
}
```

#### 6.2 Integration Tests

**Integration Test Setup (`pkg/integration/kubernetes_test.go`):**

```go
//go:build integration

package integration

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// skipIfNoKubernetes skips the test if no Kubernetes cluster is available.
func skipIfNoKubernetes(t *testing.T) {
    t.Helper()

    // Check if kubectl is available.
    if _, err := exec.LookPath("kubectl"); err != nil {
        t.Skip("kubectl not found in PATH")
    }

    // Check if cluster is accessible.
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "kubectl", "cluster-info")
    if err := cmd.Run(); err != nil {
        t.Skip("Kubernetes cluster not accessible")
    }
}

// TestKubernetesSyncLinuxPod tests synchronization to a Linux pod.
func TestKubernetesSyncLinuxPod(t *testing.T) {
    skipIfNoKubernetes(t)

    // Create a test namespace.
    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a simple Linux pod.
    podName := "test-linux-pod"
    deployLinuxPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Create a local temp directory with test files.
    localDir := t.TempDir()
    createTestFiles(t, localDir)

    // Create sync session.
    alphaURL := localDir
    betaURL := fmt.Sprintf("kubernetes://%s/%s:/tmp/sync", namespace, podName)

    sessionID := createSyncSession(t, alphaURL, betaURL)
    defer terminateSession(t, sessionID)

    // Wait for sync to complete.
    waitForSyncComplete(t, sessionID)

    // Verify files exist in pod.
    verifyFilesInPod(t, namespace, podName, "/tmp/sync")
}

// TestKubernetesMultiContainerPod tests that multi-container pods require
// explicit container specification.
func TestKubernetesMultiContainerPod(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a multi-container pod.
    podName := "test-multi-container"
    deployMultiContainerPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Attempt to create session WITHOUT container specified - should fail.
    localDir := t.TempDir()
    betaURL := fmt.Sprintf("kubernetes://%s/%s:/tmp/sync", namespace, podName)

    _, err := createSyncSessionWithError(t, localDir, betaURL)
    if err == nil {
        t.Fatal("expected error for multi-container pod without container specified")
    }
    if !strings.Contains(err.Error(), "multiple containers") {
        t.Errorf("expected 'multiple containers' error, got: %v", err)
    }

    // Now create session WITH container specified - should succeed.
    betaURLWithContainer := fmt.Sprintf("kubernetes://%s/%s/main:/tmp/sync", namespace, podName)
    sessionID := createSyncSession(t, localDir, betaURLWithContainer)
    defer terminateSession(t, sessionID)
}

// TestKubernetesForwarding tests port forwarding to a Kubernetes pod.
func TestKubernetesForwarding(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a pod running a simple HTTP server.
    podName := "test-http-server"
    deployHTTPServerPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Create forwarding session.
    sourceURL := fmt.Sprintf("kubernetes://%s/%s:tcp:localhost:8080", namespace, podName)
    destURL := "tcp:localhost:18080"

    sessionID := createForwardSession(t, sourceURL, destURL)
    defer terminateSession(t, sessionID)

    // Verify we can connect.
    verifyHTTPConnection(t, "http://localhost:18080")
}

// TestKubernetesContextOverride tests that CLI context flag overrides default.
func TestKubernetesContextOverride(t *testing.T) {
    skipIfNoKubernetes(t)

    // This test requires multiple contexts to be configured.
    contexts := getAvailableContexts(t)
    if len(contexts) < 2 {
        t.Skip("need at least 2 contexts for this test")
    }

    // Get the current (default) context.
    currentContext := getCurrentContext(t)
    
    // Find an alternative context that is different from current.
    var altContext string
    for _, ctx := range contexts {
        if ctx != currentContext {
            altContext = ctx
            break
        }
    }
    if altContext == "" {
        t.Skip("no alternative context available")
    }

    // Create a namespace and pod in the current context.
    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)
    podName := "test-pod"
    deployLinuxPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Test 1: Sync without --kubernetes-context should use current context.
    localDir := t.TempDir()
    createTestFiles(t, localDir)
    betaURL := fmt.Sprintf("kubernetes://%s/%s:/tmp/data", namespace, podName)
    sessionID := createSyncSession(t, localDir, betaURL)
    terminateSession(t, sessionID)

    // Test 2: Sync with explicit --kubernetes-context pointing to current context should work.
    sessionID = createSyncSessionWithContext(t, localDir, betaURL, currentContext)
    terminateSession(t, sessionID)

    // Test 3: Sync with --kubernetes-context pointing to a context where the pod doesn't exist
    // should fail with a clear error (either namespace or pod not found).
    _, err := createSyncSessionWithContextAndError(t, localDir, betaURL, altContext)
    if err == nil {
        t.Fatal("expected error when using wrong context, but got none")
    }
    errLower := strings.ToLower(err.Error())
    if !strings.Contains(errLower, "namespace") && !strings.Contains(errLower, "pod") && !strings.Contains(errLower, "not found") {
        t.Errorf("expected error about namespace or pod not found, got: %v", err)
    }
}

// Helper functions for context override test.

func getAvailableContexts(t *testing.T) []string {
    t.Helper()
    cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("failed to get contexts: %v", err)
    }
    var contexts []string
    for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
        if line != "" {
            contexts = append(contexts, line)
        }
    }
    return contexts
}

func getCurrentContext(t *testing.T) string {
    t.Helper()
    cmd := exec.Command("kubectl", "config", "current-context")
    output, err := cmd.Output()
    if err != nil {
        t.Fatalf("failed to get current context: %v", err)
    }
    return strings.TrimSpace(string(output))
}

func createSyncSessionWithContext(t *testing.T, localDir, betaURL, context string) string {
    t.Helper()
    cmd := exec.Command("mutagen", "sync", "create",
        "--kubernetes-context", context,
        localDir, betaURL)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to create sync session with context %s: %v (output: %s)", context, err, output)
    }
    return extractSessionID(t, string(output))
}

func createSyncSessionWithContextAndError(t *testing.T, localDir, betaURL, context string) (string, error) {
    t.Helper()
    cmd := exec.Command("mutagen", "sync", "create",
        "--kubernetes-context", context,
        localDir, betaURL)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("%s", strings.TrimSpace(string(output)))
    }
    return extractSessionID(t, string(output)), nil
}

// TestKubernetesWindowsDetectionWithHome tests that Windows containers are
// correctly detected even when HOME is set (common in Git Bash, MSYS, etc.).
func TestKubernetesWindowsDetectionWithHome(t *testing.T) {
    skipIfNoKubernetes(t)

    // This test requires a Windows node in the cluster.
    if !hasWindowsNode(t) {
        t.Skip("no Windows node available in cluster")
    }

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a Windows pod that has both HOME and USERPROFILE set.
    podName := "test-windows-pod"
    deployWindowsPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Create a sync session - should detect Windows correctly.
    localDir := t.TempDir()
    createTestFiles(t, localDir)

    betaURL := fmt.Sprintf("kubernetes://%s/%s:C:\\Users\\ContainerUser\\sync", namespace, podName)
    sessionID := createSyncSession(t, localDir, betaURL)
    defer terminateSession(t, sessionID)

    // Wait for sync to complete.
    waitForSyncComplete(t, sessionID)

    // Verify files using Windows path.
    verifyFilesInWindowsPod(t, namespace, podName, "C:\\Users\\ContainerUser\\sync")
}

// TestKubernetesShellLessContainer tests behavior with distroless/shell-less containers.
func TestKubernetesShellLessContainer(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a distroless pod (no shell, no tar).
    podName := "test-distroless-pod"
    deployDistrolessPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Attempt to create sync session - should fail with clear error.
    localDir := t.TempDir()
    createTestFiles(t, localDir)

    betaURL := fmt.Sprintf("kubernetes://%s/%s:/app/data", namespace, podName)
    _, err := createSyncSessionWithError(t, localDir, betaURL)
    if err == nil {
        t.Fatal("expected error for distroless container")
    }

    // Error could be about shell or tar - both are acceptable failure modes.
    if !strings.Contains(err.Error(), "shell") && !strings.Contains(err.Error(), "tar") {
        t.Logf("warning: error message should mention shell or tar limitation: %v", err)
    }
}

// TestKubernetesReadOnlyFilesystem tests error handling for read-only container filesystems.
func TestKubernetesReadOnlyFilesystem(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a pod with read-only root filesystem.
    podName := "test-readonly-pod"
    deployReadOnlyPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Attempt to create sync session - should fail with clear error.
    localDir := t.TempDir()
    createTestFiles(t, localDir)

    betaURL := fmt.Sprintf("kubernetes://%s/%s:/app/data", namespace, podName)
    _, err := createSyncSessionWithError(t, localDir, betaURL)
    if err == nil {
        t.Fatal("expected error for read-only filesystem")
    }

    // Error should mention read-only.
    errLower := strings.ToLower(err.Error())
    if !strings.Contains(errLower, "read-only") && !strings.Contains(errLower, "read only") {
        t.Errorf("expected error about read-only filesystem, got: %v", err)
    }
}

// TestKubernetesTarNotFound tests error handling when tar is not available.
func TestKubernetesTarNotFound(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a minimal pod without tar (but with shell for probing to succeed).
    podName := "test-notar-pod"
    deployNoTarPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Attempt to create sync session - should fail with clear error about tar.
    localDir := t.TempDir()
    createTestFiles(t, localDir)

    betaURL := fmt.Sprintf("kubernetes://%s/%s:/app/data", namespace, podName)
    _, err := createSyncSessionWithError(t, localDir, betaURL)
    if err == nil {
        t.Fatal("expected error when tar is not available")
    }

    // Error should mention tar.
    errLower := strings.ToLower(err.Error())
    if !strings.Contains(errLower, "tar") {
        t.Errorf("expected error about tar not found, got: %v", err)
    }
}

// TestKubernetesSyncWithSpacesInPath tests synchronization with paths containing spaces.
func TestKubernetesSyncWithSpacesInPath(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    // Deploy a Linux pod.
    podName := "test-linux-pod"
    deployLinuxPod(t, namespace, podName)
    waitForPodReady(t, namespace, podName)

    // Create a local directory with spaces in the name.
    localDir := t.TempDir()
    localSubDir := filepath.Join(localDir, "my data folder")
    if err := os.MkdirAll(localSubDir, 0755); err != nil {
        t.Fatalf("failed to create directory with spaces: %v", err)
    }
    testFile := filepath.Join(localSubDir, "test file.txt")
    if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
        t.Fatalf("failed to create test file: %v", err)
    }

    // Create a sync session with a path containing spaces on the remote side.
    // Note: The remote path has spaces to test shell escaping.
    betaURL := fmt.Sprintf("kubernetes://%s/%s:/tmp/sync data", namespace, podName)
    sessionID := createSyncSession(t, localDir, betaURL)
    defer terminateSession(t, sessionID)

    waitForSyncComplete(t, sessionID)

    // Verify that the file with spaces was synced correctly.
    verifyRemoteFile(t, namespace, podName, "/tmp/sync data/my data folder/test file.txt")
}

// TestKubernetesErrorMessages tests that error messages are user-friendly.
func TestKubernetesErrorMessages(t *testing.T) {
    skipIfNoKubernetes(t)

    namespace := "mutagen-test-" + randomString(8)
    defer cleanupNamespace(t, namespace)
    createNamespace(t, namespace)

    testCases := []struct {
        name          string
        betaURL       string
        errorContains string
    }{
        {
            name:          "non-existent namespace",
            betaURL:       "kubernetes://non-existent-ns-12345/pod:/tmp/data",
            errorContains: "namespace",
        },
        {
            name:          "non-existent pod",
            betaURL:       fmt.Sprintf("kubernetes://%s/non-existent-pod-12345:/tmp/data", namespace),
            errorContains: "pod",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            localDir := t.TempDir()
            _, err := createSyncSessionWithError(t, localDir, tc.betaURL)
            if err == nil {
                t.Fatal("expected error but got none")
            }
            if !strings.Contains(strings.ToLower(err.Error()), tc.errorContains) {
                t.Errorf("expected error containing %q, got: %v", tc.errorContains, err)
            }
        })
    }
}

// TestKubernetesWorkdirWithoutShell tests that workdir requirement without shell gives clear error.
func TestKubernetesWorkdirWithoutShell(t *testing.T) {
    // This is a unit test that can run without a cluster.
    transport := &kubernetesTransport{
        pod:                "test-pod",
        effectiveNamespace: "default",
        containerProbed:    true,
        containerIsWindows: false,
        containerHasShell:  false,
        containerHomeDirectory: "/home/user",
    }

    // Requesting a different working directory than home should fail.
    _, err := transport.command("test", "/different/path")
    if err == nil {
        t.Fatal("expected error for workdir without shell")
    }
    if !strings.Contains(err.Error(), "shell") {
        t.Errorf("error should mention shell: %v", err)
    }

    // Using home directory as workdir should succeed (no cd needed).
    _, err = transport.command("test", "/home/user")
    if err != nil {
        t.Errorf("should succeed with home as workdir: %v", err)
    }
}

// Helper functions for integration tests.

func hasWindowsNode(t *testing.T) bool {
    t.Helper()
    cmd := exec.Command("kubectl", "get", "nodes", "-l", "kubernetes.io/os=windows", "-o", "name")
    output, err := cmd.Output()
    if err != nil {
        return false
    }
    return strings.TrimSpace(string(output)) != ""
}

func deployWindowsPod(t *testing.T, namespace, name string) {
    t.Helper()
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  nodeSelector:
    kubernetes.io/os: windows
  containers:
  - name: main
    image: mcr.microsoft.com/windows/servercore:ltsc2022
    command: ["powershell", "-Command", "Start-Sleep -Seconds 3600"]
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy Windows pod: %v", err)
    }
}

func deployDistrolessPod(t *testing.T, namespace, name string) {
    t.Helper()
    // Use the Kubernetes pause image which is a minimal static binary with no shell/tar.
    // This image is designed to run indefinitely and is guaranteed to work.
    // Alternative: gcr.io/google-containers/pause:3.2 or registry.k8s.io/pause:3.9
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: registry.k8s.io/pause:3.9
    # The pause image runs /pause by default, no command override needed.
    # It's a static binary (~700KB) with no shell, tar, or other utilities.
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy distroless pod: %v", err)
    }
}

func deployReadOnlyPod(t *testing.T, namespace, name string) {
    t.Helper()
    // Deploy a pod with a read-only root filesystem.
    // Uses alpine with securityContext.readOnlyRootFilesystem: true.
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: alpine:3.18
    command: ["sleep", "3600"]
    securityContext:
      readOnlyRootFilesystem: true
    # Need a writable /tmp for the process to run, but agent install
    # targets /root which is read-only.
    volumeMounts:
    - name: tmp
      mountPath: /tmp
  volumes:
  - name: tmp
    emptyDir: {}
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy read-only pod: %v", err)
    }
}

func deployNoTarPod(t *testing.T, namespace, name string) {
    t.Helper()
    // Deploy a minimal pod with shell but without tar.
    // Uses busybox which has shell but we remove tar explicitly.
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: busybox:1.36
    command: ["sh", "-c", "rm -f /bin/tar && sleep 3600"]
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy no-tar pod: %v", err)
    }
}

func createNamespace(t *testing.T, name string) {
    t.Helper()
    cmd := exec.Command("kubectl", "create", "namespace", name)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to create namespace: %v", err)
    }
}

func cleanupNamespace(t *testing.T, name string) {
    t.Helper()
    cmd := exec.Command("kubectl", "delete", "namespace", name, "--ignore-not-found")
    cmd.Run() // Ignore errors during cleanup.
}

func deployLinuxPod(t *testing.T, namespace, name string) {
    t.Helper()
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: alpine:latest
    command: ["sleep", "infinity"]
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy pod: %v", err)
    }
}

func deployMultiContainerPod(t *testing.T, namespace, name string) {
    t.Helper()
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: alpine:latest
    command: ["sleep", "infinity"]
  - name: sidecar
    image: alpine:latest
    command: ["sleep", "infinity"]
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy pod: %v", err)
    }
}

func waitForPodReady(t *testing.T, namespace, name string) {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    cmd := exec.CommandContext(ctx, "kubectl", "wait", "--for=condition=Ready",
        fmt.Sprintf("pod/%s", name), "-n", namespace, "--timeout=120s")
    if err := cmd.Run(); err != nil {
        t.Fatalf("pod did not become ready: %v", err)
    }
}

// randomString generates a cryptographically random hex string.
// Uses crypto/rand to ensure uniqueness across test runs and parallel execution.
// Returns a string of the specified length (uses length/2 bytes of randomness).
func randomString(length int) string {
    bytes := make([]byte, (length+1)/2)
    if _, err := rand.Read(bytes); err != nil {
        // Fall back to timestamp-based if crypto/rand fails (shouldn't happen).
        return fmt.Sprintf("%x", time.Now().UnixNano())[:length]
    }
    return hex.EncodeToString(bytes)[:length]
}

// createTestFiles creates a set of test files in the specified directory.
func createTestFiles(t *testing.T, dir string) {
    t.Helper()
    files := map[string]string{
        "file1.txt": "content1",
        "file2.txt": "content2",
        "subdir/file3.txt": "content3",
    }
    for name, content := range files {
        path := filepath.Join(dir, name)
        if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
            t.Fatalf("failed to create directory: %v", err)
        }
        if err := os.WriteFile(path, []byte(content), 0644); err != nil {
            t.Fatalf("failed to write file: %v", err)
        }
    }
}

// createSyncSession creates a sync session and returns its unique name.
// Uses a unique name per invocation to avoid collisions in parallel tests.
func createSyncSession(t *testing.T, alpha, beta string) string {
    t.Helper()
    // Generate unique session name to avoid collisions.
    sessionName := fmt.Sprintf("test-sync-%s", randomString(8))
    cmd := exec.Command("mutagen", "sync", "create", alpha, beta, "--name="+sessionName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to create sync session: %v\noutput: %s", err, output)
    }
    return sessionName
}

// createSyncSessionWithError attempts to create a sync session and returns any error.
// If creation unexpectedly succeeds, it terminates the session to avoid leaks.
func createSyncSessionWithError(t *testing.T, alpha, beta string) (string, error) {
    t.Helper()
    // Generate unique session name in case creation succeeds.
    sessionName := fmt.Sprintf("test-sync-%s", randomString(8))
    cmd := exec.Command("mutagen", "sync", "create", alpha, beta, "--name="+sessionName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("%v: %s", err, output)
    }
    // Unexpected success - terminate to avoid leak and return the name.
    // Caller should check for nil error and handle accordingly.
    terminateCmd := exec.Command("mutagen", "sync", "terminate", sessionName)
    terminateCmd.Run()
    return sessionName, nil
}

// createForwardSession creates a forwarding session and returns its unique name.
// Uses a unique name per invocation to avoid collisions in parallel tests.
func createForwardSession(t *testing.T, source, dest string) string {
    t.Helper()
    // Generate unique session name to avoid collisions.
    sessionName := fmt.Sprintf("test-fwd-%s", randomString(8))
    cmd := exec.Command("mutagen", "forward", "create", source, dest, "--name="+sessionName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to create forward session: %v\noutput: %s", err, output)
    }
    return sessionName
}

// terminateSession terminates a session by name.
func terminateSession(t *testing.T, name string) {
    t.Helper()
    cmd := exec.Command("mutagen", "sync", "terminate", name)
    cmd.Run() // Ignore errors during cleanup.
    cmd = exec.Command("mutagen", "forward", "terminate", name)
    cmd.Run() // Ignore errors during cleanup.
}

// waitForSyncComplete waits for a sync session to complete initial synchronization.
// Fails fast if the session enters an error state.
func waitForSyncComplete(t *testing.T, name string) {
    t.Helper()
    // Poll session status until synchronized.
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    // Error state patterns that indicate permanent failure.
    errorPatterns := []string{
        "Halted",
        "Error:",
        "unable to",
        "failed to",
        "connection refused",
    }
    
    for {
        select {
        case <-ctx.Done():
            // Get final status for debugging.
            cmd := exec.Command("mutagen", "sync", "list", name)
            output, _ := cmd.Output()
            t.Fatalf("timeout waiting for sync to complete; last status:\n%s", output)
        case <-time.After(2 * time.Second):
            cmd := exec.Command("mutagen", "sync", "list", name)
            output, err := cmd.Output()
            if err != nil {
                continue
            }
            outputStr := string(output)
            
            // Check for success.
            if strings.Contains(outputStr, "Watching for changes") {
                return // Sync complete.
            }
            
            // Check for error states to fail fast.
            for _, pattern := range errorPatterns {
                if strings.Contains(outputStr, pattern) {
                    t.Fatalf("sync session entered error state:\n%s", outputStr)
                }
            }
        }
    }
}

// verifyFilesInPod verifies that expected files exist in the pod.
func verifyFilesInPod(t *testing.T, namespace, pod, path string) {
    t.Helper()
    cmd := exec.Command("kubectl", "exec", "-n", namespace, pod, "--", "ls", "-la", path)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to list files in pod: %v\noutput: %s", err, output)
    }
    // Check for expected files.
    if !strings.Contains(string(output), "file1.txt") {
        t.Error("file1.txt not found in pod")
    }
}

// verifyFilesInWindowsPod verifies files in a Windows pod.
func verifyFilesInWindowsPod(t *testing.T, namespace, pod, path string) {
    t.Helper()
    cmd := exec.Command("kubectl", "exec", "-n", namespace, pod, "--", "cmd", "/c", "dir", path)
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("failed to list files in Windows pod: %v\noutput: %s", err, output)
    }
    if !strings.Contains(string(output), "file1.txt") {
        t.Error("file1.txt not found in Windows pod")
    }
}

// verifyHTTPConnection verifies HTTP connectivity to a URL with retry/backoff.
// Forwarding sessions and pod servers often need a few seconds to become ready.
func verifyHTTPConnection(t *testing.T, url string) {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    var lastErr error
    // Retry with exponential backoff: 500ms, 1s, 2s, 4s, 8s (capped), 8s, ...
    // Total timeout is 30s via context.
    backoff := 500 * time.Millisecond
    maxBackoff := 8 * time.Second
    
    for {
        select {
        case <-ctx.Done():
            t.Fatalf("failed to connect to %s after retries: %v", url, lastErr)
        default:
        }
        
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            t.Fatalf("failed to create request: %v", err)
        }
        
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            lastErr = err
            time.Sleep(backoff)
            if backoff < maxBackoff {
                backoff *= 2
            }
            continue
        }
        
        // Check status before closing body.
        statusOK := resp.StatusCode == http.StatusOK
        
        // Always close body to prevent FD/connection leak.
        // Drain body first to enable connection reuse.
        _, _ = io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
        
        if statusOK {
            return // Success.
        }
        
        lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
        time.Sleep(backoff)
        if backoff < maxBackoff {
            backoff *= 2
        }
    }
}

// deployHTTPServerPod deploys a pod running a simple HTTP server.
func deployHTTPServerPod(t *testing.T, namespace, name string) {
    t.Helper()
    manifest := fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: python:3-alpine
    command: ["python", "-m", "http.server", "8080"]
    ports:
    - containerPort: 8080
`, name, namespace)

    cmd := exec.Command("kubectl", "apply", "-f", "-")
    cmd.Stdin = strings.NewReader(manifest)
    if err := cmd.Run(); err != nil {
        t.Fatalf("failed to deploy HTTP server pod: %v", err)
    }
}
```

#### 6.3 CI Integration

**GitHub Actions workflow addition (`.github/workflows/kubernetes-integration.yml`):**

```yaml
name: Kubernetes Integration Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Create kind cluster
        uses: helm/kind-action@v1
        with:
          cluster_name: mutagen-test

      - name: Build Mutagen
        run: go build ./...

      - name: Run Kubernetes integration tests
        run: go test -tags=integration -v ./pkg/integration/... -run TestKubernetes
```

---

### Phase 7: Documentation

#### 7.1 README Updates

Add to README.md:

```markdown
## Kubernetes Support

Mutagen supports synchronization and forwarding to containers running in
Kubernetes clusters.

### URL Format

```
kubernetes://[user@]namespace/pod[/container]:/path
```

**Note:** Context is specified via `--kubernetes-context` flag, not in the URL.

### Examples

```bash
# Sync to a pod in a namespace
mutagen sync create ./local kubernetes://default/my-pod:/app/data

# Sync to a specific container in a multi-container pod
mutagen sync create ./local kubernetes://prod/my-pod/main:/data

# Use a specific context
mutagen sync create --kubernetes-context=prod-cluster \
    ./local kubernetes://default/my-pod:/data

# Forward a port from a pod
mutagen forward create \
    kubernetes://default/my-pod:tcp:localhost:8080 \
    tcp:localhost:8080
```

### Flags

| Flag | Description |
|------|-------------|
| `--kubernetes-kubeconfig` | Path to kubeconfig file (default: `$KUBECONFIG` or `~/.kube/config`) |
| `--kubernetes-context` | Kubernetes context to use |
| `--kubernetes-namespace` | Namespace override (takes precedence over URL) |
| `--kubernetes-container` | Container override (takes precedence over URL) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `KUBECONFIG` | Path(s) to kubeconfig file(s), captured at session creation |
| `MUTAGEN_KUBECTL_PATH` | Override path to kubectl binary |

### Multi-Container Pods

For pods with multiple containers, you must specify the container either in
the URL or via the `--kubernetes-container` flag:

```bash
# In URL
mutagen sync create ./local kubernetes://default/my-pod/my-container:/data

# Via flag
mutagen sync create --kubernetes-container=my-container \
    ./local kubernetes://default/my-pod:/data
```
```

---

## Implementation Order

| Phase | Component | Estimated Time | Notes |
|-------|-----------|----------------|-------|
| 1 | Proto & URL Parsing | 4-5 hours | Includes host parsing, validation, environment capture |
| 2 | Kubectl Package | 3-4 hours | Command wrapper, platform-specific paths, shell escaping |
| 3 | Transport Layer | 8-10 hours | Core transport, probing, copy, error classification, lifecycle handling |
| 4 | Protocol Handlers | 2-3 hours | Sync and forwarding handlers |
| 5 | CLI Integration | 3-4 hours | Flags, conflict resolution, parameter application |
| 6 | Testing | 12-16 hours | Unit tests, integration tests, CI setup, path escaping tests |
| 7 | Documentation | 3-4 hours | README, help text, examples, limitations |
| 8 | Validation & Polish | 6-8 hours | Proto regen, downstream rebuilds, edge case fixes, Windows testing |

**Total Estimated Time: 50-60 hours**

### MVP Scope Definition

The initial release (MVP) targets the following scope:

**Fully Supported (MVP):**
- Linux container pods with shell (`/bin/sh`) and tar available
- Validated kubeconfig and context configuration
- POSIX filesystem paths without non-standard characters
- Single-node kind clusters for CI testing

**Experimental/Limited Support:**
- **Windows containers**: Marked as experimental; basic functionality tested but edge cases may exist
- **Distroless/shell-less containers**: Working directory changes not supported; agent must run from home directory
- **Containers without tar**: Not supported for sync (forwarding may work)

**Post-MVP Enhancements:**
- Streamed copy fallback for containers without tar (`kubectl exec cat > file`)
- Improved Windows container support
- Multi-cluster and cross-context operations
- Pod-to-pod synchronization

### Implementation Notes

This revised estimate accounts for:
- Shell escaping for paths with spaces and special characters (POSIX and Windows)
- Extended error classification (namespace, context, kubeconfig, cluster connectivity)
- Pod/container lifecycle handling (restart detection, agent reinstallation triggers)
- Tar availability and read-only filesystem error detection
- Robust environment variable handling (filtering, not just appending)
- Proper working directory support via shell wrapping with fallback for shell-less containers
- URL/flag conflict resolution logic
- Full probe parity with Docker transport (UTF-8 validation, user/group matching)
- Windows container detection even when HOME is set alongside USERPROFILE
- Shell availability probing and graceful degradation for distroless images
- Multi-container pod detection and error messaging
- Integration test infrastructure (kind cluster setup, CI gating)
- Additional tests for shell-less containers and path escaping edge cases
- Documentation of tar/shell requirements and known limitations
- Test hardening: unique session IDs, proper cleanup, retry/backoff patterns
- Robust test fixtures (working distroless images, error state detection)

---

## Files Summary

### New Files (~23 files)

| Path | Description |
|------|-------------|
| `pkg/url/parse_kubernetes.go` | URL parsing and host component parsing |
| `pkg/url/parse_kubernetes_test.go` | URL parsing tests with round-trip validation |
| `pkg/kubernetes/doc.go` | Package documentation |
| `pkg/kubernetes/kubernetes.go` | kubectl command wrapper |
| `pkg/kubernetes/kubernetes_darwin.go` | macOS-specific paths |
| `pkg/kubernetes/kubernetes_posix.go` | POSIX-specific paths |
| `pkg/kubernetes/kubernetes_windows.go` | Windows-specific paths |
| `pkg/kubernetes/kubernetes_test.go` | Command wrapper tests |
| `pkg/kubernetes/flags.go` | Connection flags handling |
| `pkg/kubernetes/flags_test.go` | Flags tests with round-trip validation |
| `pkg/agent/transport/kubernetes/doc.go` | Package documentation |
| `pkg/agent/transport/kubernetes/transport.go` | Transport implementation with shell escaping and extended error classification |
| `pkg/agent/transport/kubernetes/transport_test.go` | Transport tests including shell escaping, error classification, and lifecycle handling |
| `pkg/agent/transport/kubernetes/environment.go` | Environment filtering, `findEnvironmentVariable` helper |
| `pkg/agent/transport/kubernetes/environment_test.go` | Environment handling and variable parsing tests |
| `pkg/synchronization/protocols/kubernetes/doc.go` | Package documentation |
| `pkg/synchronization/protocols/kubernetes/protocol.go` | Sync protocol handler |
| `pkg/forwarding/protocols/kubernetes/doc.go` | Package documentation |
| `pkg/forwarding/protocols/kubernetes/protocol.go` | Forward protocol handler |
| `cmd/external/kubernetes.go` | CLI flags and URL parameter application |
| `pkg/integration/kubernetes_test.go` | Integration tests including Windows and distroless cases |
| `.github/workflows/kubernetes-integration.yml` | CI workflow for integration tests |

### Modified Files (~8 files)

| Path | Description |
|------|-------------|
| `pkg/url/url.proto` | Add Kubernetes protocol enum |
| `pkg/url/url.go` | Protocol validation/formatting for Kubernetes |
| `pkg/url/parse.go` | Add Kubernetes URL detection and dispatch |
| `pkg/url/format.go` | Add Kubernetes URL formatting |
| `cmd/mutagen/sync/create.go` | Register Kubernetes flags, apply parameters |
| `cmd/mutagen/sync/main.go` | Register flags with command |
| `cmd/mutagen/forward/create.go` | Register Kubernetes flags, apply parameters |
| `cmd/mutagen/forward/main.go` | Register flags with command |

---

## Key Considerations

### Container Requirements

**Shell Availability:**
- POSIX containers: Requires `/bin/sh` for working directory support via shell wrapping
- Windows containers: Requires `cmd.exe` for working directory support
- Shell-less containers (distroless, scratch-based): Partially supported
  - Working directory changes are not possible without a shell
  - If the agent binary location equals the home directory, no shell is needed
  - Clear error message is provided if shell is required but not available

**Tar Availability (for `kubectl cp`):**
- `kubectl cp` uses `tar` internally to copy files to/from containers
- This is a known kubectl limitation, not a Mutagen limitation
- Containers without `tar` will fail during agent installation
- **Fallback consideration:** Future enhancement could use streamed copy via `kubectl exec cat > file` pattern
- Most standard container images (alpine, debian, ubuntu, etc.) include `tar`
- Distroless images typically do NOT include `tar` - these are not supported for sync

### Error Handling

| Error Condition | Behavior |
|-----------------|----------|
| Kubeconfig not found/invalid | Clear error with path guidance, no retry |
| Context not found | Clear error suggesting --kubernetes-context, no retry |
| Cluster not reachable | Clear error about connectivity, no retry |
| Namespace not found | Clear error with namespace name, no retry |
| Pod not found | Clear error message, no retry |
| Pod not running | Clear error message, no retry |
| Pod restarted/deleted | Trigger agent reinstallation |
| Container not found | Error with container name, no retry |
| Multi-container ambiguity | Error suggesting `--kubernetes-container` flag |
| RBAC forbidden | Clear error about permissions |
| No shell available | Clear error if working directory needed, otherwise fallback to direct exec |
| No tar in container | Clear error suggesting image with tar, no retry |
| Read-only filesystem | Clear error about read-only container, no retry |
| Command not found (126/127) | Trigger agent install |
| Transient network issues | Handled by session retry logic |

**Error Detection Notes:**
- Error fragment matching (e.g., "not found", "forbidden") may vary across kubectl versions
- Exit codes are checked first when available for more reliable detection
- Error messages are defensive and check multiple patterns where possible
- Classification order matters: cluster/context errors checked before pod/container errors
- Multiple fragments are checked for common error types to reduce false negatives:
  - Cluster connectivity: `"connection refused"`, `"timeout"`, `"i/o timeout"`
  - Kubeconfig: `"kubeconfig"`, `"unable to load"`, `"invalid configuration"`
  - Context: `"context"` + `"not found"` or `"does not exist"`
  - Namespace: `"namespace"` + `"not found"`
  - Pod not found: `"not found"`, `"NotFound"`
  - Pod lifecycle: `"container has been restarted"`, `"has been deleted"`
  - Multi-container: `"must specify a container"`, `"Defaulting container"`
  - Forbidden: `"forbidden"`, `"Forbidden"`, `"unauthorized"`
  - Tar missing: `"tar: not found"`, `"executable file not found"`
  - Read-only: `"read-only file system"`, `"EROFS"`
- Future improvement: Parse structured kubectl errors via `--output=json` where supported
- Exit code 1 from kubectl is generic; fragment matching is necessary for specificity

### Security

- `KUBECONFIG` environment variable is captured at session creation time
- In-cluster service account authentication is supported (via kubectl defaults)
- Pod/namespace names are validated but not sanitized (kubectl will reject invalid names)
- User field in URL is validated against container user but not enforced by kubectl
- Shell escaping prevents command injection via path names

### Windows Containers

**Status: Experimental**

Windows container support is provided on a best-effort basis:
- Detection now handles containers with both HOME and USERPROFILE set
- First checks for USERPROFILE in POSIX `env` output, then confirms with `echo %OS%`
- If OS contains "windows", uses USERPROFILE and Windows path handling
- USERPROFILE environment variable used for home directory
- Path escaping uses Windows-style double-quote wrapping
- Path separators handled appropriately in Copy and Command methods
- Windows containers in Kubernetes are rare; edge cases may exist

### Multi-container Pods

- If container not specified and pod has multiple containers, probing fails with clear error
- Error message suggests using `--kubernetes-container` flag or URL format
- Container can be specified in URL (`namespace/pod/container`) or via CLI flag
- CLI flag takes precedence over URL

### Working Directory

- `kubectl exec` does not support `--workdir` flag
- Working directory is implemented by wrapping commands with `cd <dir> && <cmd>`
- **Path escaping:** All paths are shell-escaped to handle spaces and special characters
- POSIX with shell: Uses `sh -c "cd '/path with spaces' && command"`
- POSIX without shell: Falls back to direct command execution (no cd)
- Windows: Uses `cmd /c "cd /d "C:\path with spaces" && command"`
- Shell availability is probed during container inspection

### Environment Variable Handling

- `KUBECONFIG` is captured from the environment at parse time
- Variables are filtered (replaced, not appended) to prevent environment leakage
- `findEnvironmentVariable` helper correctly parses environment blocks
- `environment.ParseBlock` returns `[]string`, not `map[string]string`
- Handles CRLF output from Windows `cmd /c set` commands

---

## Usage Examples

```bash
# Basic sync to a pod
mutagen sync create ./local kubernetes://default/my-pod:/app/data

# Sync to a specific container in a multi-container pod
mutagen sync create ./local kubernetes://prod/my-pod/app-container:/data

# Use a specific kubeconfig and context
mutagen sync create \
    --kubernetes-kubeconfig=/path/to/kubeconfig \
    --kubernetes-context=prod-cluster \
    ./local kubernetes://default/my-pod:/data

# Override namespace from URL with flag
mutagen sync create \
    --kubernetes-namespace=staging \
    ./local kubernetes://default/my-pod:/data  # Uses "staging" not "default"

# Forward a port from a pod to local
mutagen forward create \
    kubernetes://default/my-pod:tcp:localhost:8080 \
    tcp:localhost:8080

# Forward with container specified
mutagen forward create \
    kubernetes://default/my-pod/nginx:tcp:localhost:80 \
    tcp:localhost:8080

# Sync with explicit user (for ownership validation)
mutagen sync create ./local kubernetes://appuser@default/my-pod:/home/appuser/data
```
