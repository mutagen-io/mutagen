# Contributing

Mutagen welcomes community contributions, especially feedback and experience
reports. See below for more information on the best channels for each of these.


## Issues

Issues are best submitted via the
[issue tracker](https://github.com/mutagen-io/mutagen/issues). If you're
reporting a security issue (or even just something that you *think* might be a
security issue), then please follow responsible disclosure practices and submit
the issue report using the instructions found in the
[security documentation](SECURITY.md).


## Experience reports

Experience reports are an essential part of improving Mutagen. These reports
might include problems you've had, use cases that aren't covered by existing
features, or even just general thoughts about how to improve Mutagen. All of
this also applies to Mutagen's website, documentation, and community portals.
You can send your feedback via the
[Mutagen Community Slack Workspace](https://mutagen.io/slack), submit it via the
[issue tracker](https://github.com/mutagen-io/mutagen/issues), or even just
email us at [hello@mutagen.io](mailto:hello@mutagen.io).


## Pull requests

Before taking the time to implement a change or feature, please discuss the
proposed change on the
[issue tracker](https://github.com/mutagen-io/mutagen/issues) or
[Mutagen Community Slack Workspace](https://mutagen.io/slack).

If it *does* make sense to open a pull request, please adhere to the following
guidelines. Pull requests that don't follow these guidelines will be closed.


### Developer Certificate of Origin

Pull requests to Mutagen must be submitted under the terms of the
[Developer Certificate of Origin (DCO)](DCO). In order to accept a pull request,
we require that you sign-off all commits in the pull request using the `-s` flag
with `git commit` to indicate that you agree to the terms of the DCO. You must
also
[cryptographically sign](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits)
your commits to verify your DCO sign-off.


### Code guidelines

In order to ensure that Mutagen's codebase remains clean and understandable to
all developers, we kindly request that:

- Code adheres to Go style guidelines, including those in
  [Effective Go](https://go.dev/doc/effective_go) and the
  [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- All code be `go fmt`'d
- New code matches the style and structure of its surrouding code (unless a full
  refactor/rewrite of a package is being performed)
- Comments be wrapped at 80 columns (with exceptions for long strings like URLs
  that can't be wrapped)
    - Code does not need to be wrapped at 80 lines, but please do try to keep
      lines to a reasonable length
- Comments be used to break up code blocks and be composed of full and complete
  sentences
  ([example](https://github.com/mutagen-io/mutagen/blob/da724cc1946ff70b9734be3bc5f3aae35c818c99/pkg/synchronization/core/scan.go#L142-L240))
- Imports be grouped by module, with standard library packages in the first
  group ([example](https://github.com/mutagen-io/mutagen/blob/da724cc1946ff70b9734be3bc5f3aae35c818c99/cmd/mutagen/forward/create.go#L3-L25))
- Non-trivial changes include full test coverage


### Commit guidelines

To help keep source control logs readable and useful, we also ask that all
commits have well-formatted commit messages that follow the
[Go commit message guidelines](https://go.dev/doc/contribute#commit_messages),
with no line exceeding 72 characters in length.

Here's an example of a good message:

    sync: modified controller state locking

    This commit modifies the synchronization controller's state locking to
    take into account changes that can occur during shutdown. It requires
    that the synchronization Goroutine hold the state lock until fully
    terminated.

    Fixes #00000

    Signed-off-by: Jacob Howard <jacob@mutagen.io>

Here's an example of a not-so-good message:

    fixes sync
