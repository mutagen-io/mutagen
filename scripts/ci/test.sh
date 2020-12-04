#!/bin/bash

# Exit immediately on failure.
set -e

# Load test parameters.
source scripts/ci/test_parameters.sh

# Perform a local build so that we have an agent bundle for integration tests.
go run scripts/build.go --mode=local

# Run tests and generate a coverage profile.
go test -p 1 -v -coverpkg=./pkg/... -coverprofile=coverage.txt ./pkg/...

# Run tests with the race detector enabled. We use a slim end-to-end test since
# the race detector significantly increases the execution time.
MUTAGEN_TEST_END_TO_END="slim" go test -p 1 -race ./pkg/...
