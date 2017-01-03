# Mutagen

TODO: Add CI badges.

Mutagen is a cross-platform, continuous, bi-directional file synchronization
utility designed to be simple, robust, and performant.

**Warning:** Mutagen is a very powerful tool that is still in early beta. It has
[known issues](https://github.com/havoc-io/mutagen/issues) and will almost
certainly have unknown issues. It should not be used on production or
mission-critical systems. Use on *any* system is at your own risk (please see
the [license](https://github.com/havoc-io/mutagen/blob/master/LICENSE.md)).


## Usage

For usage information, please see the [documentation site](http://mutagen.io).


## FAQs

Please see the [FAQ](https://github.com/havoc-io/mutagen/blob/master/FAQ.md).


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
- Mutagen attempts to handle quirks by default, e.g. dealing with HFS's
  pseudo-NFD Unicode normalization, systems that don't support executability
  bits, or file names that might create NTFS alternate data streams.


## Building

Please see the
[build instructions](https://github.com/havoc-io/mutagen/blob/master/BUILDING.md).
