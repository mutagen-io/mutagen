# SSH

Mutagen provides support for synchronizing with filesystem locations accessible
via SSH. This support extends to SSH clients and servers found on both POSIX and
Windows systems.


## Requirements

Mutagen requires an OpenSSH client installation to be available on your system.
This is the default for almost all POSIX operating systems, so you probably
don't need to do anything if you're on macOS, Linux, or one of the BSDs. Windows
is supported as well, but please see the [section on Windows support](#windows)
for more detailed information.

Mutagen should support most SSH server implementations, so long as they support
OpenSSH's `scp` and `ssh` implementations. If you run into trouble with a
particular SSH server implementation, please file an issue.


## Usage

SSH synchronization endpoints can be specified to Mutagen's `create` command
using an SCP-like URL syntax of the form:

    [user@]host[:port]:path

The `user` component is optional and will be passed to OpenSSH to override the
default username.

The `host` component can be any IP address, hostname, or alias understood by
OpenSSH.

The `port` component is also optional and is an extension to the standard SCP
URL syntax. If specified, it will be passed to OpenSSH to override the default
port.

The `path` component is interpreted in the same way as an SCP path. It can be an
absolute path, e.g.

    user@host:/var/www

a relative path (in which case it's considered relative to the user's home
directory), e.g.

    user@host:path/in/home/directory

a home-directory-relative path, e.g.

    user@host:~/path/in/home/directory

or an alternate user home-directory-relative path, e.g.

    user@host:~otheruser/path/in/their/home/directory

The `path` component is not allowed to be empty.


## Mechanism of action

Mutagen's SSH support is provided by the OpenSSH installation on your system.
Mutagen uses the OpenSSH suite's `scp` command to copy agent binaries to remote
systems and the `ssh` command to run and communicate with the agent binaries
over standard input/output streams.

Mutagen redirects SSH prompts to its `create` and `resume` commands via the
`SSH_ASKPASS` environment variable.

This design has a number of advantages:

- OpenSSH installations are nearly universal in the POSIX world, and
  [OpenSSH is now making its way to Windows](https://blogs.msdn.microsoft.com/powershell/2017/12/15/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/).
- Mutagen automatically uses all existing SSH configuration, keys, and aliases.
- There's no risk of baking in an SSH library that might be found to have
  security vulnerabilities.
- With OpenSSH being the *de facto* SSH implementation, other SSH server
  implementations are far more likely to aim for compatibility with OpenSSH
  clients.
- OpenSSH has a much longer develoment and security track record than any SSH
  library implementation.

Mutagen may eventually support other SSH clients or embed the
[Go SSH library](https://godoc.org/golang.org/x/crypto/ssh), but these will
always be fallback options.


## Windows

Windows is fully supported by Mutagen, though you'll need to bring your own
OpenSSH client. Unfortunately the
[official Windows OpenSSH client](https://blogs.msdn.microsoft.com/powershell/2017/12/15/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/)
is still in beta and has a few blocking issues (primarily
[1152](https://github.com/PowerShell/Win32-OpenSSH/issues/1152)) that disallow
its use by Mutagen at the moment. As soon as these issues are resolved, this
will become Mutagen's recommended client (though others will continue to be
supported).

Mutagen has a hardcoded set of OpenSSH clients that it will look for and use at
the moment, including those from
[Git for Windows](https://gitforwindows.org/) (recommended),
[MSYS2](http://www.msys2.org/), and [Cygwin](https://www.cygwin.com/). You can
see the full list of search paths
[here](https://github.com/havoc-io/mutagen/blob/fa4fa0dfe1aa35ec4c7a1c432593fd38191401da/pkg/ssh/ssh_windows.go#L13).

One thing to be aware of is that the MSYS2 and Cygwin OpenSSH clients will, by
default, look for SSH configuration (`~/.ssh`) in the "virtual" home directory
provided by the underlying Cygwin environment. This is totally fine, though
Mutagen will continue to store its data (`~/.mutagen`) and look for its
configuration (`~/.mutagen.toml`) in the user's actual home directory (i.e. that
specified by `%USERPROFILE%`). The Git for Windows OpenSSH implementation will
look for SSH configuration in the actual home directory as well, which is why
Mutagen will give it priority if found.


### SSH servers

Mutagen is currently only known to work reliably with
[Bitvise SSH Server](https://www.bitvise.com/ssh-server) on Windows.

The
[official Windows OpenSSH server](https://blogs.msdn.microsoft.com/powershell/2017/12/15/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/)
is still in beta and has a few performance issues and other kinks that need to
be ironed out before Mutagen can reliably support it. Mutagen will connect to
this server and mostly work, but the performance is not great at the moment due
to the server's stream handling. Issues have also been experienced with stalled
data streams.

There is no reason that Mutagen won't work with other Windows SSH servers (e.g.
Cygwin's OpenSSH server), so please report your findings if you try them out.


### PuTTY

Mutagen does not support [PuTTY](https://www.putty.org/). I like PuTTY, and I
tried to implement this, but it doesn't seem like PuTTY provides comparable
mechanisms for forwarding prompts and streaming standard input/output. This
isn't too surprising, as the "TTY" in its name clearly aligns with its focus on
interactive terminal usage. I know that PuTTY is widely used, but I'm hoping
that the offical Windows OpenSSH implementation will provide an easy migration
path in the very near future.
