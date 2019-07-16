# Contributing

Mutagen welcomes community contributions in all forms, including issue feedback,
experience reports, and pull requests. See below for more information on the
best channels for each of these.


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
[Mutagen community chat](https://spectrum.chat/mutagen/general), submit it via
the [issue tracker](https://github.com/mutagen-io/mutagen/issues), or even just
email us at [hello@mutagen.io](mailto:hello@mutagen.io).


## Pull requests

Mutagen is happy to receive pull requests and we want to make that procedure as
painless as possible. To that end, we've outlined a few guidelines to help make
the process go smoothly.


### Developer Certificate of Origin

Pull requests to Mutagen are submitted under the terms of the
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

Here's an example of a not-so-good message:

    fixes sync


### Just a heads up...

Please be aware that Mutagen is still at a rapid stage of development, so pull
requests may be put on the back burner if they conflict with ongoing refactors.
If you have questions or an idea for a pull request, please reach out on the
[Mutagen community chat](https://spectrum.chat/mutagen/development) before
investing a large amount of time writing code. It may be the case that someone
else is already working on the same thing!
