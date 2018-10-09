# Version control systems

Mutagen is designed to work in tandem with version control systems (VCSs),
allowing you to, e.g., clone and edit a project while mirroring it to a remote
system and testing it as your make edits. This helps you to avoid needing a
push/pull cycle every time you make a change that you want to test.

When using Mutagen with a VCS repository, there are a few "best practices" of
which you should be aware.


## VCS directories should not be synced

VCS directories (e.g. `.git`, `.svn`, `.hg`, etc.) *can* be synchronized by
Mutagen like any other directory, but they *shouldn't* be for a number of
reasons. The reasons are more or less the same for each VCS, but we'll cover the
common case of a `.git` directory. These reasons are also not specific to
Mutagen — they apply to any file synchronization tool or service.

The first reason is that the Git index data structure (which resides in `.git`)
records inode numbers, device ids, and modification times that are specific to
the filesystem on which it resides. If you move it to another system, then the
next time you run `git status` (or any command relying on similar Git
infrastructure), Git is going to have to do a full re-hash of the working tree
and will then write a new copy of the index with the inode numbers, device ids,
and modification times for the working tree on which it was just run. This is
fine if you just want to move a Git repository once, since you'll just incur a
little extra penalty the first time you run `git status`, but it won't play well
with constant synchronization. The Git index can also be a bit large (up to tens
of MB for very large working trees) and is rewritten every time you run certain
Git commands (e.g. `git status`), so you'd be constantly resynchronizing it.

The second reason is that Git's object store is not homogenous or immutable.
Some objects are stored as loose objects and some are stored in pack files, and
it will be completely dependent on the history of a particular copy of a Git
repository. They can also be pruned or relocated into pack files at any point by
Git's garbage collection. This will not play well with synchronization for a
variety of reasons that are a bit too numerous to go into, but it will be more
than a performance nuisance like the Git index — it may actually cause Git to
complain about duplicate objects, or cause weird behavior when Git does its
garbage collection. Again, this doesn't matter when you're just copying a Git
repository one time, since in that case you're not continuing to synchronize
against it.

A third reason is that Git isn't expecting concurrent modifications of its
`.git` directory. In fact it has an index lock that has to be held by Git
processes specifically for this reason.

There are a number of other reasons, but it basically comes down to the fact
that only Git is in a position to be in control of what's in its `.git` folder
(at least when it comes to the index and object stores).


## Recommended workflow

The recommended workflow for using Mutagen with VCS repositories is to
[ignore VCS directories](ignores.md#vcs), keeping a copy of the VCS directory on
only one side of the synchronization session and synchronizing only the working
tree around it. You can think of the side with the VCS directory as the "master"
and the side without as the "slave", even though the synchronization is
bidirectional. You can even have multiple "slaves" with a hub-spoke model. With
this model, you can invoke VCS commands on the "master" side (usually your
actual workstation) and, if any changes are made to the working tree, those
changes will be synchronized out to the "slaves".


## Additional workflows to avoid

In addition to avoiding direct synchronization of VCS directories, there are
other setups that are also probably best avoided.

For example, you could imagine a set up with a Git repository where you
synchronize two copies of the repository, each with a `.git` directory, but
exclude the `.git` directories from synchronization. At first this will appear
to work, because both will show you the same result for `git status` and
`git diff`. However, as soon as you do a `git commit` operation on one side, the
other side will still be on the previous commit and see modified files while the
committed side will show a clean working tree (even though both repositories
have the exact same files in their working trees). The only way to propagate the
commit to the other side would be to do a push/pull cycle, but to pull the
commit you'd need to stash your working tree changes, which would revert your
working directory back to the previous commit, which in the mean time would be
sync'd back to the committed side, which would then show modifications, at least
until you pulled down the commit on the other side, etc., etc. This is
theoretically safe, but it is very clunky and likely to cause confusing
behavior.
