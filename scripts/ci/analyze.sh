#!/bin/bash

# We want all commands to run, but we want to fail the script if any of them
# fail, so we track the exit status of each command instead of using set -e.
FAILURE=0

# Verify that code is formatted and simplified according to Go standards. We
# skip this check on Windows due to gofmt normalizing Go code to LF endings (and
# thus generating an enormous and pointless diff).
if [[ "$(go env GOOS)" != "windows" ]]; then
    gofmt -s -l -d . | tee gofmt.log
    if [[ -s gofmt.log ]]; then
        FAILURE=1
    fi
    rm gofmt.log
fi

# Perform static analysis.
go vet ./pkg/... || FAILURE=1
go vet ./cmd/... || FAILURE=1
go vet ./scripts/... || FAILURE=1

# TODO: Add spell checking. The https://github.com/client9/misspell tool is what
# we've used historically (via Go Report Card), but it seems like it's no longer
# maintained and it's installation is a little non-trivial.

# TODO: Add custom code/comment structure validation.

# Done.
exit $FAILURE
