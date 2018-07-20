# SSH

Mutagen's SSH support is provided by the OpenSSH installation on your system.
This design has a number of advantages:

- OpenSSH installations are nearly universal in the POSIX world, and
  [OpenSSH is now making its way to Windows](https://blogs.msdn.microsoft.com/powershell/2017/12/15/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/).
- Mutagen automatically uses all existing SSH configuration, keys, and aliases.
- There's no risk of baking in an SSH library that might be found to have
  security vulnerabilities.
- As the *de facto* SSH implementation, SSH servers are far more likely to
  aim for compatibility with OpenSSH.
- OpenSSH has a much longer develoment and security track record than any SSH
  library implementation.

Mutagen may eventually support other SSH clients or embed the
[Go SSH library](https://godoc.org/golang.org/x/crypto/ssh), but these will
always be fallback options.


## Mechanism of action

Mutagen uses the OpenSSH suite's `scp` and `ssh` commands to copy agent binaries
to remote systems and execute them. Communication with the agent happens over
standard input/output streams.

Mutagen redirects SSH prompts to its `create` and `resume` commands via the
`SSH_ASKPASS` environment variable.


## Recommended settings

Although not required, you might find it useful to enable OpenSSH's
[`ControlMaster` support](https://developer.rackspace.com/blog/speeding-up-ssh-session-creation/).
This speeds up SSH session creation (for all programs using OpenSSH) and, if
you're using password-based authentication, will reduce the number of times that
you'll need to type your password, especially when Mutagen connects to a system
for the first time.


## Windows

Windows is fully supported by Mutagen, though you'll need to bring your own
OpenSSH client. Unfortunately the
[official Windows OpenSSH client](https://blogs.msdn.microsoft.com/powershell/2017/12/15/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/)
is still in beta and has a few blocking issues (primarily
[1152](https://github.com/PowerShell/Win32-OpenSSH/issues/1152)) that disallow
its use in Mutagen at the moment. As soon as these issues are resolved, this
will become Mutagen's recommended client (though others will continue to be
supported).

Mutagen has a hardcoded set of OpenSSH clients that it will look for and use at
the moment, including those from
[Git for Windows](https://gitforwindows.org/) (recommended),
[MSYS2](http://www.msys2.org/), and [Cygwin](https://www.cygwin.com/).

One thing to be aware of is that the MSYS2 and Cygwin OpenSSH clients will, by
default, look for SSH configuration (`~/.ssh`) in the "virtual" home directory
provided by the underlying Cygwin environment. This is totally fine, though
Mutagen will continue to store its data (`~/.mutagen`) and look for its
configuration (`~/.mutagen.toml`) in the user's actual home directory (i.e. that
specified by `%USERPROFILE%`). The Git for Windows OpenSSH implementation will
look for SSH configuration in the actual home directory as well, which is why
Mutagen will give it priority if found.

Although Mutagen will work fine from inside a `cmd.exe`-based shell, even when
using one of these Cygwin-based OpenSSH implementations, you may wish to use
Mutagen from inside the corresponding mintty-based shell. That is also fine, but
please see the note about mintty shells in the
[platform guide](platforms.md#windows).


### SSH servers

Mutagen works with both the
[official Windows OpenSSH server](https://blogs.msdn.microsoft.com/powershell/2017/12/15/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/) and
[Bitvise SSH Server](https://www.bitvise.com/ssh-server). Please be aware that
the former is still in beta.

There is no reason that Mutagen won't work with other Windows SSH servers (e.g.
Cygwin's OpenSSH server), so please report your findings if you try them out.


### PuTTY

Mutagen does not support [PuTTY](https://www.putty.org/). I like PuTTY, and I
tried to implement this, but it doesn't seem like PuTTY provides the same
mechanisms for forwarding prompts and streaming standard input/output that
OpenSSH provides. This isn't too surprising, as the "TTY" in its name clearly
aligns with its focus on interactive terminal usage. I know that PuTTY is widely
used, but I'm hoping that the offical Windows OpenSSH implementation will
provide an easy migration path in the very near future.
