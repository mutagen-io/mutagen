#!/bin/bash

# Exit immediately on failure.
set -e

# Load test parameters.
source scripts/ci/test_parameters.sh

# Perform a local build so that we have an agent bundle for integration tests.
if [[ "${MUTAGEN_TEST_ENABLE_SSPL}" == "true" ]]; then
    go run scripts/build.go --mode=local --sspl
else
    go run scripts/build.go --mode=local
fi

# Run tests and generate a coverage profile.
if [[ "${MUTAGEN_TEST_ENABLE_SSPL}" == "true" ]]; then
    go test -tags mutagensspl -p 1 -v -coverpkg=./pkg/... -coverprofile=coverage.txt ./pkg/...
else
    go test -p 1 -v -coverpkg=./pkg/... -coverprofile=coverage.txt ./pkg/...
fi

# Run tests with the race detector enabled. We use a slim end-to-end test since
# the race detector significantly increases the execution time.
if [[ "${MUTAGEN_TEST_ENABLE_SSPL}" == "true" ]]; then
    MUTAGEN_TEST_END_TO_END="slim" go test -tags mutagensspl -p 1 -race ./pkg/...
else
    MUTAGEN_TEST_END_TO_END="slim" go test -p 1 -race ./pkg/...
fi

# Run tests on SSPL code. We perform this test on all platforms, regardless of
# whether or not SSPL-licensed enhancements are enabled for regular tests.
go test -v -tags mutagensspl ./sspl/...
