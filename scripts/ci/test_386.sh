#!/bin/bash

# Exit immediately on failure.
set -e

# Load test parameters.
source scripts/ci/test_parameters.sh

# Disable Docker tests for 386.
export MUTAGEN_TEST_DOCKER="false"

# Perform a local-only build so that we have an agent bundle for testing.
if [[ "${MUTAGEN_TEST_ENABLE_SSPL}" == "true" ]]; then
    GOARCH=386 go run scripts/build.go --mode=local --sspl
else
    GOARCH=386 go run scripts/build.go --mode=local
fi

# Run tests.
if [[ "${MUTAGEN_TEST_ENABLE_SSPL}" == "true" ]]; then
    GOARCH=386 go test -tags sspl -p 1 ./pkg/...
else
    GOARCH=386 go test -p 1 ./pkg/...
fi

# Run tests on SSPL code. We perform this test on all platforms, regardless of
# whether or not SSPL-licensed enhancements are enabled for regular tests.
GOARCH=386 go test -v -tags sspl ./sspl/...
