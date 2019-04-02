# Mutagen

Mutagen is a **fast**, continuous, multidirectional file synchronization tool.
It can safely, scalably, and efficiently synchronize filesystem contents between
arbitrary pairs of locations in near real-time. Support is currently implemented
for locations on local filesystems, SSH-accessible filesystems, and Docker
container filesystems.

Mutagen is designed specifically to support remote development scenarios, with
configurable behaviors specifically designed to help developers edit code
locally while building, running, or packaging it in a remote environment.

For more information, please see [https://mutagen.io](https://mutagen.io).


## Usage

For an overview of Mutagen's design and basic usage, please see the
["Design and usage" guide](https://mutagen.io/documentation/design-and-usage/).

Information about more advanced topics is also available in
[Mutagen's documentation](https://mutagen.io/documentation).


## Status

Mutagen is a very powerful tool that is still in early beta. It will almost
certainly have unknown issues. It should not be used on production or
mission-critical systems. Use on *any* system is at your own risk (please see
the [license](https://github.com/havoc-io/mutagen/blob/master/LICENSE)).

That being said, Mutagen is a very useful tool and I use it daily for work on
remote systems. The more people who use it and report
[issues](https://github.com/havoc-io/mutagen/issues), the better it will get!

| Windows                           | macOS/Linux                                   | Code coverage                           | Report card                           |
| :-------------------------------: | :-------------------------------------------: | :-------------------------------------: | :-----------------------------------: |
| [![Windows][win-badge]][win-link] | [![macOS/Linux][mac-lin-badge]][mac-lin-link] | [![Code coverage][cov-badge]][cov-link] | [![Report card][rc-badge]][rc-link]   |

[win-badge]: https://ci.appveyor.com/api/projects/status/qywidv5a1vf7g3b5/branch/master?svg=true "Windows build status"
[win-link]:  https://ci.appveyor.com/project/havoc-io/mutagen/branch/master "Windows build status"
[mac-lin-badge]: https://travis-ci.org/havoc-io/mutagen.svg?branch=master "macOS/Linux build status"
[mac-lin-link]:  https://travis-ci.org/havoc-io/mutagen "macOS/Linux build status"
[cov-badge]: https://codecov.io/gh/havoc-io/mutagen/branch/master/graph/badge.svg "Code coverage status"
[cov-link]: https://codecov.io/gh/havoc-io/mutagen/tree/master/pkg "Code coverage status"
[rc-badge]: https://goreportcard.com/badge/github.com/havoc-io/mutagen "Report card status"
[rc-link]: https://goreportcard.com/report/github.com/havoc-io/mutagen "Report card status"


## Community

To follow release and security announcements for Mutagen, please subscribe to
the [mutagen-announce](https://groups.google.com/forum/#!forum/mutagen-announce)
mailing list.

For discussion about Mutagen usage, please join the
[discussion forums](https://groups.google.com/forum/#!forum/mutagen).


## Contributing

If you'd like to contribute to Mutagen, please see the
[contribution documentation](CONTRIBUTING.md).


## Security

Mutagen takes security very seriously. If you believe you have found a security
issue with Mutagen, please practice responsible disclosure practices and send an
email directly to [security@mutagen.io](mailto:security@mutagen.io) instead of
opening a GitHub issue. For more information, please see the
[security documentation](SECURITY.md).


## Building

Please see the [build instructions](BUILDING.md).
