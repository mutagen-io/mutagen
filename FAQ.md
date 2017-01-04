# Frequently asked questions

This list is a lie - none of these questions have been asked, not even
infrequently. If you have a question or concern that I haven't addressed, please
open an issue.


## Usage

- **Is there a GUI?** Not yet, but one is in development. This should negate the
  need to manually launch the daemon, as well as provide grapical session
  management and monitoring. Expect this circa late February or early March.
- **Can I synchronize more than two endpoints together?** Yes, though not
  directly. You'll want to set up a star topology with one copy of the files at
  the center (probably on your local machine, though technically it doesn't have
  to be) that's synchronized with multiple remote copies via separate sessions.
- **How do I resolve a conflict?** Delete the root of the conflict on the
  endpoint that you want to lose. If you need to merge the changes that have
  occurred on each endpoint in a conflict, then you'll have to do this manually.
- **Can I ignore paths?** Yes, you can use the `-i/--ignore` flag in the
  `create` command. Support at the moment is very basic, but covers most cases.
  Patterns are matched using the
  [doublestar](https://github.com/bmatcuk/doublestar) package, with the
  additional feature that ignores can be negated with a `!` prefix in order to
  override any previous ignore. For example, you can specify
  `mutagen create --ignore="foo*" --ignore="!foobar" ...`, which will ignore
  contents with names like "foofoo" or "foofoobar" but not "foobar". It is not
  possible to unignore a child path of a directory that has been matched by a
  previous ignore, because the filesystem scanner will not even descend into
  ignored directories. Also, all ignore paths are treated as relative to the
  synchronization root, they are not evaulated within each directory. If you
  want to evaluate an ignore within each directory, prefix it with a doublestar,
  e.g. `mutagen create --ignore="**/.git" ...` will ignore any directories named
  ".git". Ignored paths will not be deleted by Mutagen if their parents are
  deleted on the remote side - you'll have to manually delete them. This is by
  design.
- **Does it support symlinks?** Only for synchronization roots and even then
  with some caveats. If a symlink is used as a synchronization root, it will
  work (e.g. a symlink to a directory), but if the root is replaced or removed
  by the synchronization algorithm (e.g. it's a directory that's deleted or a
  file that's modified), then the symlink itself will be replaced. Within
  synchronization roots, symlinks are simply ignored. It may be possible to add
  support for them in the future, especially with the Windows symlink support
  [finally being brought into the spotlight](https://blogs.windows.com/buildingapps/2016/12/02/symlinks-windows-10/),
  but from a theoretical standpoint, there's only two ways to handle them:
  either recreate them (in which case they may point to nothing or even be
  invalid if their path doesn't translate) or copy a file with the path of the
  symlink in it (which is pretty much useless and just prone to messing up the
  actual symlink). So they're ignored for now. They won't be deleted either,
  meaning that if you delete a directory hierarchy and the change propagates to
  a replica of that directory that contains a symlink, it will be marked as a
  problem requiring manual resolution (i.e. it will require you to manually
  delete the symlink). This is by design.
- **Can I use it for respositories?** Yes! This was actually why I originally
  created Mutagen - the desire to develop cross-platform applications without
  having to write code in the console or a VM or having to push/pull changes to
  test every two seconds. You can have a single copy of the repository that you
  edit (in your nice $70 text editor) and then mirror to your test platforms
  (which you might then interact with using the terminal). If you do this, it's
  highly recommended that you ignore SCM directories (e.g. `.git`, `.svn`, or
  `.hg`) using the ignore patterns described above. Although Mutagen WILL
  synchronize these, the index files that modern SCMs use for quick re-scans can
  be large and will be rewritten (with many changes) every time you do something
  like `git status`. By ignoring `.git` and its ilk, you also add an insurance
  policy that it won't be deleted by Mutagen if the mirror of the repository is
  deleted on the remote.
- **Does it work on Windows?** Yes! But it only supports OpenSSH, not PuTTY. At
  the moment, it only supports Cygwin-based OpenSSHs (e.g. those provided by
  [Cygwin](https://www.cygwin.com/), [MSYS2](https://msys2.github.io/), or
  [Git for Windows](https://git-scm.com/)), but it *will* support the
  [PowerShell team's OpenSSH port](https://github.com/PowerShell/Win32-OpenSSH)
  once that's released. The PowerShell port currently has some significant bugs
  that prevent Mutagen from operating.
- **Will you add proper packaging?** Yes, eventually I want to have Windows
  installers, macOS... somethings (not .pkg files, maybe Homebrew), and
  .deb/.rpm packages, ideally with a PPA or similar. At the moment though,
  development is a bit too heavy to be pushing out those types of changes. Also,
  the distribution only consists of two files, so it's not too painful. I'd be
  particular interested in Windows packaging help if someone is a WiX guru.
- **What platforms are supported?** Ideally the same platforms supported by the
  [current version of Go](https://golang.org/doc/install/source#environment). A
  few platforms (e.g. Plan 9, Android, and iOS) are currently disabled because
  their ports are either flimsy, they are missing necessary OS facilities, or
  they don't make any sense to support. You can look at the build script
  (`scripts/build.go`) to see the platforms for which binaries are currently
  being built. If something is missing, please let me know. Please note that I
  don't have the facilities to personally test every OS and architecture
  combination, so if you have a system that doesn't seem to work, open an issue
  and we can work on fixing things.
- [**Mutagen is broken... can you make it go?**](https://www.youtube.com/watch?v=-WmGvYDLsj4)
  Hopefully! Open an issue and let's have a look.


## Design

- **Why do you only support OpenSSH? Why not use the Git SSH library?** OpenSSH
  is really the defacto SSH implementation - everything aims to be compatible
  with it. By relying on it, we get a robust, well-tested transport. It is also
  one of the only SSH clients that allows for passwords to be provided securely
  by another program. By not embedding our own SSH library, we keep binary size
  down, remove the need to put out new versions when CVEs arise in the Go SSH
  implementation, and generally get all the extensive configuration support that
  OpenSSH provides (which a large number of people use for non-trivial
  configuration).
- **Why don't you store or at least cache passwords?** Most platforms that
  provide secure password storage do so via a C library, so adding support would
  significantly complicate our binaries. Moreover, these libraries are generally
  quite varied in terms of interface. This is something I looked into and even
  [started writing support for](https://github.com/havoc-io/go-keytar), but it
  was more trouble than it was worth. It's also very difficult (if not
  impossible) to determine when stored/cached passwords should be invalidated.
  I understand that some organizations disable public key authentication, and in
  those cases I would recommend
  [enabling SSH ControlMaster support](https://developer.rackspace.com/blog/speeding-up-ssh-session-creation/)
  to make your life more bearable.
- **How does it work?** The synchronization algorithm is fairly simple, but I
  haven't had time to document it yet. This is coming though! The essential idea
  is to watch each endpoint for changes, make a metadata snapshot of the
  synchronization root on each endpoint, reconcile changes using a three-way
  merge with an ancestor snapshot, stage changes, apply changes, and then update
  the ancestor snapshot with the changes that successfully propagated. The
  synchronization routines are in the `sync` package. They are put to use in the
  `session` package.
- **Why Go? Rust makes me feel safer.** Yeah, me too. Go is currently the only
  language that has the requisite cross-compiling capabilities, syscall-only
  binaries, and simple asynchronous I/O handling. It does have some downsides
  though, e.g. binary size, memory usage, thread usage (exacterbated by IPC in
  our case), and most importantly its inability to enforce certain invariants in
  synchronization data structures and algorithms. A rewrite in Rust is not out
  of the question once rustup and tokio mature, especially since Mutagen is
  < 15 KLOC.
