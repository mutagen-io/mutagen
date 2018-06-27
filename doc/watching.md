# Filesystem watching

Mutagen uses filesystem watching to know when it should re-scan and propagate
files. Unfortunately, the filesystem watching landscape is *extremely* varied in
terms of implementation, efficiency, and robustness. Almost every platform uses
a completely different mechanism, many of which are unreliable or non-scalable.
For example, some systems (namely macOS and Windows) provide recursive watching
mechanisms that can monitor an arbitrarily large directory hierarchy, but their
behavior when the location being watched is deleted or changed is problematic or
useless. The "watch a parent directory" solution quickly becomes a
turtles-all-the-way-down situation, with no way to ever guarantee the existence
or lifetime of the watched location. Other systems (e.g. Linux and BSD systems)
provide mechanisms that require a watch descriptor or file descriptor to be open
for *every* file being watched in a directory hierarchy, which can quickly
exhaust system quotas for directories that might be used in development (e.g.
imagine a synchronization root containing a `node_modules` directory).

Mutagen takes a pragmatic but potentially controversial approach to filesystem
watching that deserves documenting. If a synchronization root is on a system
that supports efficient recursive watching, and the synchronization root is a
subdirectory of the user's home directory, then a watch is established on the
home directory and events are filtered to only those originating from the
synchronization root. This side-steps the issue of the watch root being deleted
(if your home directory is deleted, you have other problems). In all other
cases, a polling mechanism is used to avoid exhausting watch/file descriptors.

Obviously watching one's home directory is a matter of taste. If you want to
review this behavior, the code is in
[`watch_native_recursive.go`](https://github.com/havoc-io/mutagen/blob/master/pkg/filesystem/watch_native_recursive.go).
If that doesn't calm your nerves, then fear not, because you can disable it.

Mutagen provides two different filesystem watching modes:

- **Portable** (Default): In this mode, Mutagen uses the most efficient watching
  implementation possible for an endpoint. If a synchronization root meets the
  requirements for the recursive home directory watching method, then that is
  used, otherwise poll-based watching is used.
- **Force Poll**: In this mode, Mutagen will always use its poll-based watching
  implementation.

These options are obviously not ideal or exhaustive. Active R&D is underway to
improve the situation and provide additional filesystem watching options, so
please stand by. If you have any feedback on other efficient (but portable and
safe) filesystem watching designs, please feel free to contact me or open an
issue to discuss your proposal.

These modes can be specified on a per-session basis by passing the
`--watch-mode=<mode>` flag to the `create` command (where `<mode>` is `portable`
or `force-poll`) and on a default basis by including the following configuration
in `~/.mutagen.toml`:

    [watch]
    mode = "<mode>"

The polling interval (which defaults to 10 seconds) can be specified on a
per-session basis by passing the `--watch-polling-interval=<interval>` flag to
the `create` command (where `<interval>` is an integer value representing
seconds) and on a default basis by including the following configuration in
`~/.mutagen.toml`:

    [watch]
    pollingInterval = <interval>
