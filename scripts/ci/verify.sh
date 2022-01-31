#!/bin/bash

# Exit immediately on failure.
set -e

# Print status information.
echo "Performing commit verification"
echo

# Track that we actually perform some sort of verification, because we don't
# have an elegant way to verify that git rev-list succeeds when being used via
# process substitution.
PERFORMED_VERIFICATION="false"

# Loop over the relevant commits.
while read commit; do
    # Print status information.
    echo "> Verifying ${commit}"

    # Verify that the expected sign-off is present.
    EXPECTED_SIGNOFF="$(git show "${commit}" --format="format:Signed-off-by: %an <%ae>" --no-patch)"
    if git show --summary "${commit}" | grep -q "${EXPECTED_SIGNOFF}"; then
        echo "Found valid sign-off"
    else
        echo "Missing sign-off!"
        exit 1
    fi

    # Verify that a cryptographic signature is present.
    # TODO: It may be worth trying to corresponding GPG keys from GitHub to
    # verify that they match the commit author, but that's going to be tricky.
    # We'll also have to consider the possibility that the signatures were made
    # with OpenSSH keys, and I'm not sure how to import those for Git-based
    # verification. For now, we can just check the verified label on the GitHub
    # web interface.
    if [[ ! -z "$(git show --format="format:%GK" --no-patch "${commit}")" ]]; then
        echo "Found cryptographic signature"
    else
        echo "Missing or invalid cryptographic signature!"
        exit 1
    fi

    # TODO: Validate the commit message format.

    # Record that some verification was performed.
    PERFORMED_VERIFICATION="true"

    # Output a separator line.
    echo
done < <(git rev-list "${VERIFY_COMMIT_START}...${VERIFY_COMMIT_END}")

# Enforce that at least one commit was verified.
if [[ "${PERFORMED_VERIFICATION}" == "false" ]]; then
    echo "No verification performed!"
    exit 1
fi

# Print status information.
echo "Commit verification succeeded!"
