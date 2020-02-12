#!/bin/sh

# Exit on any failure.
set -e

# Set the Mutagen version to use for tunnel hosting as well the default list of
# extra agent versions to install.
MUTAGEN_TUNNEL_HOST_VERSION="0.11.0"
DEFAULT_MUTAGEN_TUNNEL_AGENT_VERSIONS="0.11.0"

# Determine agent versions.
MUTAGEN_TUNNEL_AGENT_VERSIONS="${MUTAGEN_TUNNEL_AGENT_VERSIONS=${DEFAULT_MUTAGEN_TUNNEL_AGENT_VERSIONS}}"

# Determine the GOOS value.
echo "Probing OS"
UNAMES="$(uname -s)"
case "${UNAMES}" in
    "Linux")
        GOOS="linux"
        ;;
    *)
        echo "Unsupported operating system" >&2
        exit 1
        ;;
esac

# Determine the GOARCH value.
echo "Probing architecture"
UNAMEM="$(uname -m)"
case "${UNAMEM}" in
    "x86_64")
        GOARCH="amd64"
        ;;
    *)
        echo "Unsupported architecture" >&2
        exit 1
        ;;
esac

# Install Mutagen.
echo "Installing Mutagen version ${MUTAGEN_TUNNEL_HOST_VERSION}"
cd /usr/bin
curl -fsSL "https://github.com/mutagen-io/mutagen/releases/download/v${MUTAGEN_TUNNEL_HOST_VERSION}/cli_${GOOS}_${GOARCH}_v${MUTAGEN_TUNNEL_HOST_VERSION}.tar.gz" | tar xz

# Disable file globbing.
# TODO: Figure out a way to restore initial f setting here. It should be +f
# by default, but there's probably some way to detect this.
set -f

# Store the old internal field separator and set it to split versions.
OLDIFS="${IFS=_MUTAGEN_IFS_UNSET}"
IFS=":"

# Install agents.
echo "Installing agents"
MUTAGEN_TUNNEL_AGENTS_PATH="/usr/libexec/mutagen/agents"
mkdir -p "${MUTAGEN_TUNNEL_AGENTS_PATH}"
for MUTAGEN_AGENT_VERSION in $MUTAGEN_TUNNEL_AGENT_VERSIONS; do
    # Update status.
    echo "Installing agent version ${MUTAGEN_AGENT_VERSION}"

    # Compute the agent's minor version.
    MUTAGEN_AGENT_MINOR_VERSION="$(echo "${MUTAGEN_AGENT_VERSION}" | cut -d '.' -f1,2)"

    # Compute the install path.
    AGENT_INSTALL_PATH="${MUTAGEN_TUNNEL_AGENTS_PATH}/${MUTAGEN_AGENT_MINOR_VERSION}"

    # Check that there's not a duplicate minor version already installed. This
    # indicates a version misconfiguration and we should indicate it to the
    # user. Otherwise create the installation directory.
    if [ -d "${AGENT_INSTALL_PATH}" ]; then
        echo "Duplicate agent version already installed" >&2
        exit 1
    else
        mkdir "${AGENT_INSTALL_PATH}"
    fi

    # Download and install.
    cd "${AGENT_INSTALL_PATH}"
    curl -fsSL "https://github.com/mutagen-io/mutagen/releases/download/v${MUTAGEN_AGENT_VERSION}/agent_${GOOS}_${GOARCH}_v${MUTAGEN_AGENT_VERSION}.tar.gz" | tar xz
done

# Restore the internal field separator.
if [ "${OLDIFS}" = "_MUTAGEN_IFS_UNSET" ]; then
    unset IFS
else
    IFS="${OLDIFS}"
fi

# Re-enable file globbing.
set +f
