# Safety mechanisms

Mutagen has two **best-effort** safety mechanisms aimed at avoiding
unintentional synchronization root deletion. Each of these mechanisms detects an
"irregular" condition during synchronization and halts the synchronization cycle
until the user confirms that the condition is intentional.

The first feature detects complete deletion of the synchronization root on one
side of the connection. This detection is best-effort since directory deletion
is non-atomic and Mutagen may see (and propagate) deletion of a large portion of
a synchronization root before seeing that the entire root was deleted (though
Mutagen does its best to avoid operating during concurrent file modifications
when it detects them).

The second feature detects replacement of the synchronization root with a root
of a different type (e.g. replacing a directory root with a file root) on one
side of the connection.

In both cases, the user is required to delete the synchronization root on the
side that they want to delete or replace, and then use `mutagen resume` to
continue synchronization for the session.
