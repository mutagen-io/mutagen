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
        echo "Code is not go fmt'd" 1>&2
        FAILURE=1
    fi
    rm gofmt.log
fi

# Ensure that a go mod tidy operation doesn't change go.mod or go.sum. We skip
# this check on Windows due to go mod tidy normalizing module files to LF
# endings (and thus triggering a false positive).
if [[ "$(go env GOOS)" != "windows" ]]; then
    PRE_TIDY_GO_MOD_SUM=$(cat go.mod | openssl dgst -sha256)
    PRE_TIDY_GO_SUM_SUM=$(cat go.sum | openssl dgst -sha256)
    go mod tidy || FAILURE=1
    POST_TIDY_GO_MOD_SUM=$(cat go.mod | openssl dgst -sha256)
    POST_TIDY_GO_SUM_SUM=$(cat go.sum | openssl dgst -sha256)
    if [[ "${POST_TIDY_GO_MOD_SUM}" != "${PRE_TIDY_GO_MOD_SUM}" ]]; then
        echo "go.mod changed with go mod tidy operation" 1>&2
        FAILURE=1
    fi
    if [[ "${POST_TIDY_GO_SUM_SUM}" != "${PRE_TIDY_GO_SUM_SUM}" ]]; then
        echo "go.sum changed with go mod tidy operation" 1>&2
        FAILURE=1
    fi
fi

# Perform static analysis.
VETFLAGS="-composites=false"
go vet ${VETFLAGS} ./pkg/... || FAILURE=1
go vet ${VETFLAGS} ./cmd/... || FAILURE=1
go vet ${VETFLAGS} ./scripts/... || FAILURE=1
go vet ${VETFLAGS} ./tools/... || FAILURE=1

# Perform static analysis on SSPL code.
go vet -tags mutagensspl ./sspl/... || FAILURE=1

# TODO: Add spell checking. The https://github.com/client9/misspell tool is what
# we've used historically (via Go Report Card), but it seems like it's no longer
# maintained and it's installation is a little non-trivial.

# TODO: Add custom code/comment structure validation.

# Done.
exit $FAILURE
