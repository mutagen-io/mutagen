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

> **Please note:** Mutagen is currently in a state of high code churn and I
simply don't have the time to review pull requests. GitHub unfortunately doesn't
allow me to disable pull requests, but please be aware that they will probably
be left on the back burner for the foreseeable future. Sorry :-(
>
> – Jacob

Before taking the time to implement a change or feature, please discuss the
proposed change on the
[Mutagen Community Slack Workspace](https://mutagen.io/slack).

If it *does* make sense to open a pull request, please adhere to the following
guidelines. Pull requests that don't follow these guidelines will simply be
closed.


### Developer Certificate of Origin

Pull requests to Mutagen must be submitted under the terms of the
[Developer Certificate of Origin (DCO)](DCO). In order to accept a pull request,
we require that you sign-off all commits in the pull request using the `-s` flag
with `git commit` to indicate that you agree to the terms of the DCO.


### Code guidelines

In order to ensure that Mutagen's codebase remains clean and understandable to
newcomers, we kindly request that:

- Code adheres to Go style guidelines, including those in
  [Effective Go](https://golang.org/doc/effective_go.html) and the
  [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- All code be `go fmt`'d
- Comments be wrapped at 80 columns (with exceptions for long strings like URLs)
- Comments be used to break up code (see existing code for examples) and be
  composed of full and complete sentences
- New code include full test coverage

Basically, when in doubt, new code should share the same style as the
surrounding code.


### Commit guidelines

To help keep source control logs readable and useful, we also ask that all
commits have well-formatted commit messages composed of a single subject line of
50-70 characters, followed by a blank line, and finally the full, correctly
punctuated commit message (also wrapped to 80 lines). We ask the same for the
pull request message itself.

Here's an example of a good message:

    Modified synchronization controller state locking

    This commit modifies the synchronization controller's state locking to take
    into account changes that can occur during shutdown. It requires that the
    synchronization Goroutine hold the state lock until fully terminated.

    Fixes #00000

    Signed-off-by: Jacob Howard <jacob@mutagen.io>

Here's an example of a not-so-good message:

    fixes sync
