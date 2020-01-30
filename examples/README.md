# Examples

This directory contains example setups that show how to use Mutagen (in
conjunction with container orchestration tools) to create remote development
environments that work with your local tools. These examples are designed to be
used as templates for more complex setups, so they aim for simplicity while
trying to cover the most common cases.

More advanced behaviors can be achieved using Mutagen's extensive
[synchronization](https://mutagen.io/documentation/synchronization) and
[forwarding](https://mutagen.io/documentation/forwarding) configuration options,
as well as by combining multiple sessions for a single project or codebase. It's
important to remember that Mutagen works between any pair of endpoints
(local/local, local/remote, remote/local, and remote/remote), with any
combination of [transports](https://mutagen.io/documentation/transports), and
can synchronize files and forward network traffic in arbitrary directions. The
examples given here only demonstrate a very limited (but pragmatic) subset of
these behaviors.

If you need help brainstorming the right setup for your particular use case,
check out the Mutagen community chat:

[![Join the community on Spectrum](https://withspectrum.github.io/badge/badge.svg)](https://spectrum.chat/mutagen)


## Contributing

If you'd like to contribute an example of using Mutagen with your technology
stack of choice, we'd love for you to
[open a Pull Request](../CONTRIBUTING.md#pull-requests)! The easiest way to do
this might be to fork one of the existing examples and adjust certain parts of
it to meet your needs, but variety is also very welcome and helpful!
