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

    # Enforce commit message line length restrictions.
    MAXIMUM_LINE_LENGTH=$(git show --format="format:%B" --no-patch "${commit}" | wc -L)
    if [[ "${MAXIMUM_LINE_LENGTH}" -le "72" ]]; then
        echo "Commit message line length acceptable"
    else
        echo "Commit message line length too long!"
        exit 1
    fi

    # Verify that the expected sign-off is present.
    EXPECTED_SIGNOFF="$(git show "${commit}" --format="format:Signed-off-by: %an <%ae>" --no-patch)"
    if git show --format="format:%B" --no-patch "${commit}" | grep -q "${EXPECTED_SIGNOFF}"; then
        echo "Found valid sign-off"
    else
        echo "Missing sign-off!"
        exit 1
    fi

    # Verify that a cryptographic signature is present. Ideally we'd want to use
    # git-show for this, but its signature formatting simply refuses to print
    # any SSH signature information correctly (it doesn't even print %G?
    # correctly) unless the gpg.ssh.allowedSignersFile setting is set to a file
    # (even an empty one). I assume this is a bug that will be fixed in later
    # verisons of Git, but for now we'll just grab the raw commit headers and
    # check that a gpgsig header is present. We use the sed command to halt
    # git cat-file output at the first empty line, which signals the end of
    # headers, to avoid false positives from commit message text. Unfortunately
    # git-show also lacks the ability to print arbitrary raw header fields.
    #
    # TODO: It may be worth trying to corresponding GPG and/or SSH keys from
    # GitHub to verify that they match the commit author, but that's going to be
    # tricky and probably fragile. It would allow us to avoid this hack and
    # provide stronger validation, but for the time being we can likely rely on
    # GitHub account security and commit verification to provide validation.
    if [[ ! -z "$(git cat-file commit "${commit}" | sed "/^$/q" | grep "gpgsig ")" ]]; then
        echo "Found cryptographic signature"
    else
        echo "Missing or invalid cryptographic signature!"
        exit 1
    fi

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
