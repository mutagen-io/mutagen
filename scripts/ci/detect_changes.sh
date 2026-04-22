#!/bin/bash

# Exit immediately on failure.
set -e

# write_output writes a workflow output. When running locally, it falls back to
# stdout so that the script remains easy to inspect and debug.
write_output() {
    local name="$1"
    local value="$2"

    if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
        echo "${name}=${value}" >> "${GITHUB_OUTPUT}"
    else
        echo "${name}=${value}"
    fi
}

# write_plan writes the final booleans that control optional CI work.
write_plan() {
    local run_windows_docker="$1"
    local run_sidecar="$2"

    write_output "run_windows_docker" "${run_windows_docker}"
    write_output "run_sidecar" "${run_sidecar}"
}

# determine_base identifies the comparison base for change detection. Pull
# requests use their merge base with the target branch so that unrelated base
# branch changes do not affect the result.
determine_base() {
    if [[ "${MUTAGEN_CI_EVENT_NAME}" == "pull_request" ]]; then
        git merge-base HEAD "${MUTAGEN_CI_PULL_REQUEST_BASE_SHA}"
    elif [[ "${MUTAGEN_CI_EVENT_NAME}" == "merge_group" ]]; then
        echo "${MUTAGEN_CI_MERGE_GROUP_BASE_SHA}"
    else
        echo "${MUTAGEN_CI_PUSH_BASE_SHA}"
    fi
}

# has_changes checks whether any of the specified paths differ from the
# comparison base.
has_changes() {
    ! git diff --quiet "${MUTAGEN_CI_BASE}..HEAD" -- "$@"
}

# Full CI always enables all optional work, so no path-based planning is
# necessary in that case.
if [[ "${MUTAGEN_CI_FULL}" == "true" ]]; then
    echo "Full CI requested; enabling all optional work."
    write_plan "true" "true"
    exit 0
fi

# Determine and validate the comparison base. If the base cannot be resolved,
# then err on the side of running the optional work.
MUTAGEN_CI_BASE="$(determine_base)"
if [[ -z "${MUTAGEN_CI_BASE}" ]]; then
    echo "Unable to determine a comparison base; enabling all optional work."
    write_plan "true" "true"
    exit 0
fi

if ! git rev-parse --verify "${MUTAGEN_CI_BASE}^{commit}" >/dev/null 2>&1; then
    echo "Unable to resolve comparison base; enabling all optional work."
    write_plan "true" "true"
    exit 0
fi

echo "Comparing ${MUTAGEN_CI_BASE}..HEAD for optional CI work."

# Detect whether Windows Docker transport coverage should run in slim CI. The
# workflow file is included so that changes to the gating logic exercise the
# path they control.
RUN_WINDOWS_DOCKER="false"
if has_changes \
    .github/workflows/ci.yml \
    pkg/docker \
    pkg/integration \
    scripts/ci/docker \
    scripts/ci/setup_docker.sh \
    scripts/ci/test.sh \
    scripts/ci/test_parameters.sh
then
    RUN_WINDOWS_DOCKER="true"
fi

# Detect whether the sidecar image build should run in slim CI. The workflow
# file is included for the same reason as above.
RUN_SIDECAR="false"
if has_changes \
    .github/workflows/ci.yml \
    cmd/mutagen-sidecar \
    pkg/sidecar \
    images/sidecar \
    scripts/ci/sidecar_tag.go
then
    RUN_SIDECAR="true"
fi

# Write the final plan.
write_plan "${RUN_WINDOWS_DOCKER}" "${RUN_SIDECAR}"
