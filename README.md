# Mutagen

Mutagen is a cross-platform, continuous, bidirectional file synchronization
utility designed to be simple, robust, and performant. It can operate locally or
over SSH.

**Warning:** Mutagen is a very powerful tool that is still in early beta. It has
[known issues](https://github.com/havoc-io/mutagen/issues) and will almost
certainly have unknown issues. It should not be used on production or
mission-critical systems. Use on *any* system is at your own risk (please see
the [license](https://github.com/havoc-io/mutagen/blob/master/LICENSE.md)).


## Status

| Windows                           | macOS/Linux                                   | Code coverage                           | Report card                           |
| :-------------------------------: | :-------------------------------------------: | :-------------------------------------: | :-----------------------------------: |
| [![Windows][win-badge]][win-link] | [![macOS/Linux][mac-lin-badge]][mac-lin-link] | [![Code coverage][cov-badge]][cov-link] | [![Report card][rc-badge]][rc-link]   |

[win-badge]: https://ci.appveyor.com/api/projects/status/qywidv5a1vf7g3b5/branch/master?svg=true "Windows build status"
[win-link]:  https://ci.appveyor.com/project/havoc-io/mutagen/branch/master "Windows build status"
[mac-lin-badge]: https://travis-ci.org/havoc-io/mutagen.svg?branch=master "macOS/Linux build status"
[mac-lin-link]:  https://travis-ci.org/havoc-io/mutagen "macOS/Linux build status"
[cov-badge]: https://codecov.io/gh/havoc-io/mutagen/branch/master/graph/badge.svg "Code coverage status"
[cov-link]: https://codecov.io/gh/havoc-io/mutagen "Code coverage status"
[rc-badge]: https://goreportcard.com/badge/github.com/havoc-io/mutagen "Report card status"
[rc-link]: https://goreportcard.com/report/github.com/havoc-io/mutagen "Report card status"


## Usage

For usage information, please see the
[documentation site](https://havoc-io.github.io/mutagen). For platform-specific
instructions and known issues, please see the
[platform guide](doc/PLATFORMs.md).


## FAQs

Please see the [FAQ](doc/FAQ.md).


## Unique features

Rather than provide a heavily biased feature comparison table, I'll just point
out what I consider to be the unique and compelling features of Mutagen. Astute
readers with knowledge of the file synchronization landscape can draw their own
conclusions. I'd recommend users read this list so they know what they're
getting.

- Mutagen is a user-space utility, not requiring any kernel extensions or
  administrative permissions to use.
- Mutagen only needs to be installed on the computer where you want to control
  synchronization. Mutagen comes with a broad range of small, cross-compiled
  "agent" binaries that it automatically copies as necessary to remote
  endpoints. Most major platforms and architectures are supported.
- Mutagen propagates changes bi-directionally. Any conflicts that arise will be
  flagged for resolution. Resolution is performed by manually deleting the
  undesired side of the conflict. Conflicts won't stop non-conflicting changes
  from propagating.
- Mutagen uses the [rsync](https://rsync.samba.org/tech_report/) algorithm to
  perform differential file transfers. These transfers are pipelined to mitigate
  the effects of latency. It stages files outside of the synchronization root
  and relocates them atomically.
- Mutagen is designed to handle very large directory hierarchies efficiently. It
  uses rsync to transfer directory snapshots when performing reconciliation so
  that snapshot transfer time doesn't scale linearly with directory size.
- Mutagen is robust to connection drop-outs. It will attempt to reconnect
  automatically to endpoints and will resume synchronization safely. It will
  also resume staging files where it left off.
- Mutagen identifies changes to file contents rather than just modification
  times.
- On systems that support recursive file monitoring (Windows and macOS), Mutagen
  effeciently watches synchronization roots for changes. Other systems currently
  use regular and efficient polling out of a desire to support very large
  directory hierarchies that might exhaust watch file descriptors, but support
  for watching small directories on these systems isn't ruled out.
- Mutagen is agnostic of the transport to endpoints - all it requires is a byte
  stream to each endpoint. Support is currently built-in for local and SSH-based
  synchronization, but support for other remote types can easily be added. As a
  corollary, Mutagen can even synchronize between two remote endpoints without
  ever needing a local copy of the files.
- Mutagen can display dynamic synchronization status in the terminal.
- **Mutagen does not propagate (most) permissions.** Mutagen is much like Git in
  this regard - it only propagates entry type and executability. This is by
  design, since Mutagen's raison d'être is remote code editing and mirroring.
  Nothing in the current Mutagen design precludes adding permission propagation
  in the future.
- Mutagen attempts to handle quirks by default, e.g. dealing with
  case-(in)sensitivity, HFS's pseudo-NFD Unicode normalization, systems that
  don't support executability bits, or file names that might create NTFS
  alternate data streams.

You might have surmised that Mutagen's closest cousin is the
[Unison](http://www.cis.upenn.edu/~bcpierce/unison) file synchronization tool.
This tool has existed for ages, and while it is *very* good at what it does, it
didn't quite fit my needs. In particular, it has a *lot* of knobs to turn, puts
a lot of focus on transferring permissions (which can cause even more headache),
and requires installation on both ends of the connection. I wanted something
simpler, a bit more performant, and just a bit more modern (the fact that Unison
is written in rather terse OCaml also makes it a bit difficult to extend or
support on more obscure platforms and architectures).


## Building

Please see the [build instructions](doc/BUILDING.md).
