# Ignores

By default, Mutagen attempts to propagate all files that it sees within a
synchronization root. This isn't always desired, so Mutagen supports ignoring
paths within a synchronization root and excluding them from synchronization.
When a path is ignored, it won't be scanned, it won't be propagated, and it
won't be deleted.

Mutagen allows ignores to be specified on both a default and per-session basis,
and provides utilities for ignoring certain kinds of common directories.

Mutagen also provides a rich syntax for specifying ignores, which should be
familiar to Git users.


## Global ignores

Global ignores affect all newly created sessions. These should be used for files
that should always be ignored, e.g. those pesky `.DS_Store` files on macOS.

When a new session is created, global ignores are "locked in" to the session
configuration, meaning that subsequent changes to global ignores will only be
reflected in subsequently created sessions. This is for simplicity (no need to
think through the effects of changing ignores in one place) and safety (no files
will suddenly become unignored and propagated (causing conflicts or security
risks) or silently ignored).

Global ignores are specified in the `~/.mutagen.toml` as follows:

    [ignore]
    default = [
        # System files
        ".DS_Store",
        "._*",

        # Vim files
        "*~",
        "*.sw[a-p]",
    ]

Obviously you'll want to put whatever ignores in here that *you* want.


## Per-session ignores

Per-session ignores only affect the session to which they are attached. They are
processed after global ignores, meaning that they can extend or cancel out
global ignores.

To specify ignores on a per-session basis, use the `--ignore` flag of the
`create` command.


## Ignore groups

Mutagen provides ignore "groups" that provide a pre-defined set of ignores that
can be used for sessions. At the moment, only one ignore group is defined in
Mutagen, which ignores VCS directories (e.g. `.git`, `.svn`, etc.), though
additional ignore groups (e.g. for Python object code files, `node_modules`
directories, etc.) will be coming in future versions.

Ignore groups are processed before both global and per-session ignores, so they
can be extended or cancelled out by global and per-session ignores.


### VCS

VCS directories can be ignored on a per-session basis by passing the
`--ignore-vcs` flag to the `create` command and on a default basis by including
the following configuration in `~/.mutagen.toml`:

    [ignore]
    vcs = true

A default VCS ignore setting can be overridden on a per-session basis by passing
the `--no-ignore-vcs` flag to the `create` command.


## Format and behavior

The format and behavior of Mutagen's ignore patterns are designed to match those
of Git, insofar as that makes sense for Mutagen's somewhat different
application.

Ignore patterns are essentially paths that support special syntax for matching.
At a base level, ignore patterns are built using the
[doublestar](https://github.com/bmatcuk/doublestar) package, so all of the
syntax there carries over.

Ignore patterns use `/` as a path separator. All paths are treated as relative
to the synchronization root.

If an ignore pattern does not contain a `/`, i.e. it is a leaf name, then it
will also be treated as relative to any subdirectory in the synchronization
root. For example, an ignore pattern of `some/path` will only ignore content at
`<root>/some/path`, but an ignore pattern of `path` will ignore content at
`<root>/path`, `<root>/some/path`, `<root>/other/path`, etc. If you want to
ignore content that exists directly at the root and not any other subpath, then
prefix the ignore with `/`. For example, `build` will ignore content named
`build` at any subdirectory, but `/build` will only ignore content at
`<root>/build`.

Suffixing a pattern with `/` will cause it to match only directories.

Prefixing a pattern with `!` will cause it to negate previously matching
ignores. E.g. specifying an ignore list like `["hot*", "!hotel"]` will ignore
any content whose name begins with "hot", unless the full content name is
"hotel". This can be a useful mechanism for overriding global ignores on a
per-session basis.
