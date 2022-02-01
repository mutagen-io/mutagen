#!/bin/bash

# Exit immediately on failure.
set -e

# Print the Go version.
go version

# Pre-download Go modules.
go mod download
