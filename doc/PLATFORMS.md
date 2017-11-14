# Platforms

Mutagen fully supports almost all platforms supported by Go, with exceptions and
caveats listed below.


## Windows

- If using a mintty environment (e.g. Cygwin, MinGW, MSYS2, etc.), the `mutagen`
  command needs to be wrapped in a
  [`winpty`](https://github.com/rprichard/winpty) command in order to function
  properly. If run directly inside a mintty console, both
  [daemonization](https://github.com/havoc-io/mutagen/issues/3) and
  [command line prompting](https://github.com/howeyc/gopass/issues/34) will
  fail to function properly. Since the `winpty` command is present in most
  mintty environments, the best option is simply to add a shell alias, e.g.
  `alias mutagen="winpty mutagen"`.


## FreeBSD

- There are outstanding FreeBSD issues in the Go runtime which are not fully
  understood. These are documented in the
  [Go 1.9 release notes](https://golang.org/doc/go1.9#known_issues), and may
  affect the functionality and/or stability of Mutagen on FreeBSD.


## NetBSD

- NetBSD is not currently well-supported by Go due to instabilities on the
  platform and a number of outstanding issues in the Go runtime. These problems
  are documented in the
  [Go 1.9 release notes](https://golang.org/doc/go1.9#known_issues), and may
  affect the functionality and/or stability of Mutagen on NetBSD.


## Plan 9

- Plan 9 currently lacks a number of facilities necessary to build and run
  Mutagen. It *may* be possible to build `mutagen-agent` binaries for Plan 9
  (allowing you to sync files to/from Plan 9 systems), but some build
  constraints will need to be tweaked to make this work. It'll also be necessary
  to set up some sort of test platform, because Plan 9 is very different from
  POSIX systems.


## Android

- Android is currently not supported, but may be supported in the future as a
  synchronization destination (i.e. a system capable of running
  `mutagen-agent`). If you're interested in testing support for Android (I don't
  have an Android device), then please contact me.


## iOS

- iOS is not currently supported, but may be supported in the future as a
  synchronization destination (i.e. a system capable of running
  `mutagen-agent`).
