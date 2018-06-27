# Configuration

Mutagen is designed to have sane, conservative defaults, with as little manual
configuration as possible. That being said, sane defaults only go so far, and
regular users will most likely want to tweak Mutagen's behavior in one way or
another.

Mutagen has three levels of configuration: its own default behavior, the global
configuration file (`~/.mutagen.toml`), and per-session configuration.

Mutagen strives to require as little configuration as possible, so its default
behavior is designed to be sane, safe, and portable.

The global configuration file allows users to override Mutagen's default
behavior with their own defaults that will apply to all newly created sessions.
The file is a [TOML](https://github.com/toml-lang/toml) file with sections
affecting various aspects of Mutagen's behavior. Existence of the global
configuration file is *not* required.

Per-session configuration is provided by flags passed to the `create` command

Global configuration takes precendence over default behavior, and per-session
configuration takes precedence over both. When a session is created, global and
per-session configuration are read-in and merged. The merged configuration is
"locked in" to the session so that subsequent changes to the `~/.mutagen.toml`
file will not affect the behavior of existing sessions. This increases safety
while removing the cognitive load of having to understand how global
configuration changes would propagate.

Mutagen's configuration options are minimal at the moment and the goal is to
keep them that way. Configuration parameters are available for
[symlinks](symlinks.md), [ignores](ignores.md), and
[filesystem watching](watching.md).
