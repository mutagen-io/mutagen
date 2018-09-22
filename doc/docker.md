**NOTE:** Mutagen's support for Docker is considered "experimental" in the sense
that it is new (less mature than SSH support), minimalist, and designed for
experimentation. It provides the basic primitives needed to enable
synchronization with containers, but it will be up to users to integrate this
into their individual orchestration tools and workflows. However, Mutagen wants
to help with this, so the
[Docker design feedback issue](https://github.com/havoc-io/mutagen/issues/41)
has been opened for users to provide feedback on evolving Mutagen's Docker
support for easier integration. Please don't hesitate to submit your feedback on
this issue, no matter how small.


# Docker

Mutagen has support for synchronizing with filesystems inside Docker containers.
This support extends to all Docker client platforms (Linux, macOS, Windows,
etc.), Docker daemon setups (local, remote, VM, Hyper-V, etc.), and Docker
container types (both Linux and Windows containers are supported).


## Requirements

Mutagen requires the `docker` command to be in the user's path. Due to its use
of the `-w/--workdir` flag with the `docker exec` command, Mutagen requires a
Docker client and daemon supporting API 1.35+. You can check the API version
support of your Docker client and daemon by using the `docker version` command.


## Usage

Docker container filesystem endpoints can be specified to Mutagen's `create`
command using URLs of the form:

    docker://[user@]container/path

Docker URLs support Unicode names and paths and neither require nor support
URL escape encoding (i.e. just type "ö", not "%C3%B6", etc.).

The `user` component is optional and tells Mutagen to act as the specified user
inside the container. If unspecified, Mutagen uses the default container user
(usually `root` for Linux containers or `ContainerAdministrator` for Windows
containers).

The `container` component can specify any type of container identifier
understood by `docker cp` and `docker exec`, e.g. a name or hexidecimal
identifier.

The `path` component must be non-empty (i.e. at least a `/` character) and can
take one of four forms: an absolute path, a home-directory-relative path, a
home-directory-relative path for an alternate user, or a Windows absolute path:

    # Example absolute path (/var/www)
    docker://container/var/www

    # Example home-directory-relative path (~/project)
    docker://container/~/project

    # Example alternate user home-directory-relative path (~otheruser/project)
    docker://container/~otheruser/project

    # Example Windows path (C:\path)
    docker://container/C:\path

Docker containers must be running to create synchronization sessions and to
allow synchronization to run.


### Environment variables

The Docker client's behavior is controlled by three environment variables:

- `DOCKER_HOST`
- `DOCKER_TLS_VERIFY`
- `DOCKER_CERT_PATH`

Mutagen is aware of these environment variables and will lock them in at session
creation time (i.e. the `create` command will scan and store their values),
including locking in absent or empty values. These locked in values will then be
forwarded to any `docker` commands executed by Mutagen.

If required, endpoint-specific versions of these variables (prefixed with
`MUTAGEN_ALPHA_` or `MUTAGEN_BETA_`) can be used to override their values on a
per-endpoint basis. This would be necessary to (e.g.) create a synchronization
session between two Docker containers hosted on different Docker daemons. In
that case, single global values for `DOCKER_*` environment variables can't be
used (since it would apply to both endpoint URLs), and endpoint-specific
variables such as `MUTAGEN_ALPHA_DOCKER_HOST` or
`MUTAGEN_BETA_DOCKER_TLS_VERIFY` would need to be used when invoking
`mutagen create`.


### Windows containers

Docker runs Windows containers using Microsoft's Hyper-V hypervisor, which
unfortunately does not allow `docker cp` operations to copy files into running
containers. This means that Mutagen has to stop and restart Windows containers
in order to copy its agent executables. Mutagen will prompt the user if this is
necessary and allow the user to abort or proceed. If the user decides to
proceed, the stop and restart will be automatically performed by Mutagen. This
is only necessary if a compatible agent binary doesn't already exist in the
container, so it won't be necessary on subsequent connection operations.

This restriction does not apply to Linux containers.
