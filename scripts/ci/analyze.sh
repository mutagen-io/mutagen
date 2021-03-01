#!/bin/bash

# We want all commands to run, but we want to fail the script if any of them
# fail, so we track the exit status of each command instead of using set -e.
FAILURE=0

# Perform static analysis.
go vet ./pkg/... || FAILURE=1
go vet ./cmd/... || FAILURE=1
go vet ./scripts/... || FAILURE=1

# TODO: Add gofmt -s support.
# TODO: Add golint (https://github.com/golang/lint).
# TODO: Add ineffassign (https://github.com/gordonklaus/ineffassign).
# TODO: Add misspell (https://github.com/client9/misspell).
# TODO: Add custom code/comment structure validation.

# Done.
exit $FAILURE
