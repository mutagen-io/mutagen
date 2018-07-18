# Filesystem watching

Mutagen uses filesystem watching to know when it should scan for and propagate
changes. Unfortunately, the filesystem watching landscape is *extremely* varied
in terms of implementation, efficiency, and robustness. Almost every platform
uses a completely different mechanism, many of which are unreliable or
non-scalable. For example, some systems (namely macOS and Windows) provide
native recursive watching mechanisms that can monitor arbitrarily large
directory hierarchies, but their behavior when the location being watched is
deleted or changed is problematic or useless, and Windows only supports using a
directory as the root of such a recursive watch. Other systems (e.g. Linux, BSD
systems, and Solaris) provide mechanisms that require a watch descriptor or file
descriptor to be open for *every* file or directory being watched in a directory
hierarchy, which can quickly exhaust system quotas for directories that might be
used in development (e.g. imagine a synchronization root containing a
`node_modules` directory). Mutagen takes a pragmatic approach to filesystem
watching that attempts to maximize reliability and responsiveness while avoiding
exhaustion of system resources or problematic behavior.

On systems that natively support recursive filesystem watching, a watch is
established on the parent directory of the synchronization root and events are
filtered to only those originating from the synchronization root. Because these
systems can behave strangely if the root of a watch is deleted, a regular (but
very cheap) polling mechanism is used to ensure that the watch root hasn't been
deleted or recreated. If a change to the watch root is detected, the watch is
re-established.

On all other systems, a polling mechanism is used to avoid exhausting watch/file
descriptors. This polling mechanism is also used as a fallback in cases where
native watching mechanisms fail unrecoverably. On Linux, this polling is coupled
with a restricted set of native watches on the most recently updated contents,
allowing for low-latency change notifications for most workflows without
exhausting watch/file descriptors.

Mutagen provides two different filesystem watching modes:

- **Portable** (Default): In this mode, Mutagen uses the most efficient watching
  implementation possible for an endpoint. If some type of native watching
  mechanism is available, it is used, otherwise pure poll-based watching is
  used.
- **Force Poll**: In this mode, Mutagen will always use its poll-based watching
  implementation, even on systems that support native watching.

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
