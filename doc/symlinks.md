# Symlinks

Mutagen has full support for symbolic links, both on POSIX and Windows systems.
Because these platforms have very different symlink implementations, Mutagen
provides a few different symlink synchronization modes aimed at providing
maximum compatibility:

- **Ignore**: In this mode, Mutagen simply ignores any symlinks that it
  encounters within a synchronization root. This means that it won't delete them
  and it won't propagate them. This mode is supported on both POSIX and Windows
  systems.
- **Portable** (default): In this mode, Mutagen restricts itself to propagating
  only symlinks that it defines as "portable". Portable symlinks are those which
  are relative paths, containing only safe characters, which do point outside
  the synchronization root at any point in their target path. In this mode,
  Mutagen also performs appropriate symlink normalization on Windows endpoints
  to ensure that symlinks are correctly round-tripped to disk. If a non-portable
  symlink is detected within the synchronization root, the session will halt
  synchronization until it is removed or corrected. This mode is supported on
  both POSIX and Windows systems, though please see the
  [note](#windows-permissions) below about Windows permissions.
- **POSIX Raw**: In this mode, which is only supported for synchronization
  sessions between two POSIX endpoints, Mutagen will propagate raw symlink
  targets without any analysis or modification. Trying to use the POSIX Raw
  synchronization mode when either endpoint is a Windows system will stop
  synchronization from starting.

These modes can be specified on a per-session basis by passing the
`--symlink-mode=<mode>` flag to the `create` command (where `<mode>` is
`ignore`, `portable`, or `posix-raw`) and on a default basis by including the
following configuration in `~/.mutagen.toml`:

    [symlink]
    mode = "<mode>"


## Windows permissions

On Windows, the `SeCreateSymbolicLinkPrivilege` permission is required to create
symlinks. By default, this permission is usually only granted to administrators.
This has changed a bit in Windows 10, where anyone can create symlinks if
Developer Mode has been enabled and the
`SYMBOLIC_LINK_FLAG_ALLOW_UNPRIVILEGED_CREATE` flag is passed to
`CreateSymbolicLinkW`. Go has
[implemented](https://github.com/golang/go/commit/c23afa9ddb1180b929ba09a7d96710677a2a4b45)
support for passing this flag, but it won't land until Go 1.11. Mutagen will
incorporate this change as soon as Go 1.11 is released.

If you don't have the `SeCreateSymbolicLinkPrivilege` permission and can't add
it for yourself, or you can't enable Developer Mode in Windows 10 and wait for
Go 1.11, then you have two choices:

1. Do nothing. In this case, Mutagen will attempt to propagate symlinks (if it's
   in Portable mode), and will simply report that it is unable to do so. This
   won't hurt anything, and it won't stop other files from synchronizing. The
   only downside is that you'll see problems indicated when listing or
   monitoring the session.
2. Switch to ignoring symlinks.
