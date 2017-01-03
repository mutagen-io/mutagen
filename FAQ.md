# Frequently asked questions

This list is a lie - none of these questions have been asked, not even
infrequently. If you have a question or concern that I haven't addressed, please
open an issue.


## Usage

- **Is there a GUI?** Not yet, but one is in development. This should negate the
  need to manually launch the daemon, as well as provide grapical session
  management and monitoring. Expect this circa late February or early March.
- **How do I resolve a conflict?** Delete the root of the conflict on the
  endpoint that you want to lose. If you need to merge the changes that have
  occurred on each endpoint in a conflict, then you'll have to do this manually.
- **Does it work on Windows?** Yes! But it only supports OpenSSH, not PuTTY. At
  the moment, it only supports Cygwin-based OpenSSHs (e.g. those provided by
  [Cygwin](https://www.cygwin.com/), [MSYS2](https://msys2.github.io/), or
  [Git for Windows](https://git-scm.com/)), but it *will* support the
  [PowerShell team's OpenSSH port](https://github.com/PowerShell/Win32-OpenSSH)
  once that's released. The PowerShell port currently has some significant bugs
  that prevent Mutagen from operating.
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
  significantly complicate our binaries. Moreover, these libraries are general
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
  though is to watch each endpoint for changes, make a metadata snapshot of the
  synchronization root on each endpoint, reconcile changes using a three-way
  merge with an ancestor snapshot, stage changes, apply changes, and then update
  the ancestor snapshot with the changes that successfully. The synchronization
  routines are in the `sync` package. They are put to use in the `session`
  package.
- **Why Go? Rust makes me feel safer.** Yeah, me too. Go is currently the only
  language that has the requisite cross-compiling capabilities, syscall-only
  binaries, and simple asynchronous I/O handling. It does have some downsides
  though, e.g. binary size, memory usage, thread usage (exacterbated by IPC in
  our case), and most importantly its inability to enforce certain invariants in
  synchronization data structures and algorithms. A rewrite in Rust is not out
  of the question once rustup and tokio mature, especially since Mutagen is
  < 15 KLOC.
