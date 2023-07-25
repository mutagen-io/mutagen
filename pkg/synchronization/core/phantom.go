package core

// reifyPhantomDirectories is the underlying recursive implementation of
// ReifyPhantomDirectories. It performs the conjoined, reverse-DFS traversal and
// returns whether or not trackable (though not necessarily synchronizable)
// content is present at this level.
func reifyPhantomDirectories(ancestor, alpha, beta *Entry) (bool, uint64, uint64) {
	// If neither alpha nor beta is a directory kind, then we won't continue
	// recursion any further. In this case, we'll return an indication of
	// whether either alpha or beta represents non-nil, tracked content (note
	// that at least one of alpha or beta must be non-nil here). This indication
	// will tell the parents of alpha and beta whether or not to reify to
	// tracked or ignored (assuming reification is necessary). Note that we
	// don't require synchronizability to indicate that the parents should be
	// reified to tracked, because a problematic entry must be a tracked entry.
	alphaIsDirectoryKind := alpha != nil &&
		(alpha.Kind == EntryKind_Directory || alpha.Kind == EntryKind_PhantomDirectory)
	betaIsDirectoryKind := beta != nil &&
		(beta.Kind == EntryKind_Directory || beta.Kind == EntryKind_PhantomDirectory)
	if !alphaIsDirectoryKind && !betaIsDirectoryKind {
		alphaIsTrackedKind := alpha != nil && alpha.Kind != EntryKind_Untracked
		betaIsTrackedKind := beta != nil && beta.Kind != EntryKind_Untracked
		return alphaIsTrackedKind || betaIsTrackedKind, 0, 0
	}

	// At this point, we know that one or both of alpha and beta are directory
	// kinds. Thus, we need to recurse regardless of whether they're phantoms or
	// not, because they could contain phantoms at a lower depth. As we recurse,
	// we'll track whether or not tracked content exists at lower levels. Note
	// that, unlike in the reconcile case, we don't let ancestorContents drive
	// recursion, because that's only necessary to generate deletion changes to
	// the ancestor (whereas here we're not modifying the ancestor).
	ancestorContents := ancestor.GetContents()
	alphaContents := alpha.GetContents()
	betaContents := beta.GetContents()
	var trackedContentExistsAtLowerLevels bool
	var alphaDirectoryCount, betaDirectoryCount uint64
	for name := range nameUnion(alphaContents, betaContents) {
		tracked, alphaCount, betaCount := reifyPhantomDirectories(
			ancestorContents[name],
			alphaContents[name],
			betaContents[name],
		)
		if tracked {
			trackedContentExistsAtLowerLevels = true
		}
		alphaDirectoryCount += alphaCount
		betaDirectoryCount += betaCount
	}

	// Update initial counts for the current level.
	if alphaIsDirectoryKind {
		alphaDirectoryCount++
	}
	if betaIsDirectoryKind {
		betaDirectoryCount++
	}

	// Determine how to reify any phantom directories at this level. Any tracked
	// content at lower levels indicates that we should reify phantom
	// directories to tracked directories, as does the presence of a tracked
	// directory in the ancestor.
	ancestorIsDirectory := ancestor != nil && ancestor.Kind == EntryKind_Directory
	reifyToTracked := trackedContentExistsAtLowerLevels || ancestorIsDirectory
	alphaIsPhantom := alpha != nil && alpha.Kind == EntryKind_PhantomDirectory
	betaIsPhantom := beta != nil && beta.Kind == EntryKind_PhantomDirectory
	if reifyToTracked {
		if alphaIsPhantom {
			alpha.Kind = EntryKind_Directory
		}
		if betaIsPhantom {
			beta.Kind = EntryKind_Directory
		}
	} else {
		if alphaIsPhantom {
			alpha.Kind = EntryKind_Untracked
			alpha.Contents = nil
			alphaDirectoryCount--
		}
		if betaIsPhantom {
			beta.Kind = EntryKind_Untracked
			beta.Contents = nil
			betaDirectoryCount--
		}
	}

	// Return an indication of whether or not tracked content exists at or below
	// this level. Note that we still want ancestorIsDirectory folded in to this
	// return value because we know at this point that one of alpha or beta is a
	// directory type, either tracked or phantom, and in the latter case the
	// ancestorIsDirectory condition would have reified it to tracked.
	return reifyToTracked, alphaDirectoryCount, betaDirectoryCount
}

// ReifyPhantomDirectories performs a conjoined, reverse-DFS traversal of the
// ancestor, alpha, and beta entries, reifying phantom directories to either
// tracked directories or untracked content, as appropriate. It returns modified
// copies of the alpha and beta entries, along with updated directory counts for
// both alpha and beta (since these can change during directory reification
// (though note that other counts cannot)). This function is only necessary if
// Docker-style ignore syntax and semantics are being used, because phantom
// directories don't exist with standard Mutagen-style ignores.
func ReifyPhantomDirectories(ancestor, alpha, beta *Entry) (*Entry, *Entry, uint64, uint64) {
	// Create deep copies of alpha and beta that we can mutate. We won't mutate
	// any leaf entries, so we can perform a more efficient copy.
	alpha = alpha.Copy(EntryCopyBehaviorDeepPreservingLeaves)
	beta = beta.Copy(EntryCopyBehaviorDeepPreservingLeaves)

	// Perform reification.
	_, alphaDirectoryCount, betaDirectoryCount := reifyPhantomDirectories(ancestor, alpha, beta)

	// Done.
	return alpha, beta, alphaDirectoryCount, betaDirectoryCount
}
