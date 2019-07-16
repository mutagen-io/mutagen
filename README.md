# Mutagen

Mutagen offers **fast**, continuous, multidirectional file synchronization,
allowing you to work with remote code and files using your local editor and
tools in effectively real-time. Support is currently implemented for
synchronization between local filesystems, SSH-accessible filesystems, and
Docker container filesystems, with more on the way!


## Getting started

After [installing](https://mutagen.io/documentation/installation/) Mutagen,
creating a synchronization session is as simple as:

    mutagen create ~/my_project me@example.org:~/my_remote_project

Using Mutagen, you can create synchronization sessions between arbitrary pairs
of endpoints (including sessions where both endpoints are remote) and control
them from your local system. You also have granular control over the exact
synchronization behavior in case Mutagen's defaults don't meet your needs.

You can learn more in Mutagen's
[design and usage guide](https://mutagen.io/documentation/design-and-usage/) and
find information about all of Mutagen's features in
[the documentation](https://mutagen.io/documentation).


## Community

Mutagen's community chat is the place to go for discussion, questions, and
ideas:

[![Join the community on Spectrum](https://withspectrum.github.io/badge/badge.svg)](https://spectrum.chat/mutagen)

For updates about the project and its releases, you can subscribe to the
[mutagen-announce](https://groups.google.com/forum/#!forum/mutagen-announce)
mailing list or [follow the project on Twitter](https://twitter.com/mutagen_io)!


## Status

Mutagen is built and tested on Windows, macOS, and Linux, and it's available for
[many more platforms](https://github.com/mutagen-io/mutagen/releases/latest).

| Windows                           | macOS/Linux                                   | Code coverage                           | Report card                         | License                                   |
| :-------------------------------: | :-------------------------------------------: | :-------------------------------------: | :---------------------------------: | :---------------------------------------: |
| [![Windows][win-badge]][win-link] | [![macOS/Linux][mac-lin-badge]][mac-lin-link] | [![Code coverage][cov-badge]][cov-link] | [![Report card][rc-badge]][rc-link] | [![License][license-badge]][license-link] |

[win-badge]: https://ci.appveyor.com/api/projects/status/mr8rmxl5hbxgyged/branch/master?svg=true "Windows build status"
[win-link]:  https://ci.appveyor.com/project/havoc-io/mutagen-87cwp/branch/master "Windows build status"
[mac-lin-badge]: https://travis-ci.org/mutagen-io/mutagen.svg?branch=master "macOS/Linux build status"
[mac-lin-link]:  https://travis-ci.org/mutagen-io/mutagen "macOS/Linux build status"
[cov-badge]: https://codecov.io/gh/mutagen-io/mutagen/branch/master/graph/badge.svg "Code coverage status"
[cov-link]: https://codecov.io/gh/mutagen-io/mutagen/tree/master/pkg "Code coverage status"
[rc-badge]: https://goreportcard.com/badge/github.com/mutagen-io/mutagen "Report card status"
[rc-link]: https://goreportcard.com/report/github.com/mutagen-io/mutagen "Report card status"
[license-badge]: https://img.shields.io/github/license/wasmerio/wasmer.svg "MIT licensed"
[license-link]: LICENSE "MIT licensed"


## Contributing

If you'd like to contribute to Mutagen, please see the
[contribution documentation](CONTRIBUTING.md).


## External projects

Users have built a number of cool projects to extend and integrate Mutagen into
their workflows:

- [**Mutagen Helper**](https://github.com/gfi-centre-ouest/mutagen-helper) is a
  tool that makes the orchestration of synchronization sessions even easier by
  letting you define sessions with configuration files that live inside your
  codebase. Thanks to **@Toilal**!


## Security

Mutagen takes security very seriously. If you believe you have found a security
issue with Mutagen, please practice responsible disclosure practices and send an
email directly to [security@mutagen.io](mailto:security@mutagen.io) instead of
opening a GitHub issue. For more information, please see the
[security documentation](SECURITY.md).


## Building

Please see the [build instructions](BUILDING.md).
