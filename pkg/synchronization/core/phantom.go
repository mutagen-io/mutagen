package core

// reifyPhantomDirectories is the underlying recursive implementation of
// ReifyPhantomDirectories. It performs the conjoined, reverse-DFS traversal and
// returns whether or not trackable (though not necessarily synchronizable)
// content is present at this level, along with updated directory counts for
// alpha and beta, respectively.
func reifyPhantomDirectories(ancestor, alpha, beta *Entry) (bool, uint64, uint64) {
	// If neither alpha nor beta is a directory kind, then we won't continue
	// recursion any further. In this case, we'll return an indication of
	// whether either alpha or beta represents non-nil, tracked content. This
	// indication will tell the parents of alpha and beta whether or not to
	// reify to tracked or ignored (assuming reification is necessary). Note
	// that we don't require synchronizability to indicate that the parents
	// should be reified to tracked, because a problematic entry is implicitly a
	// tracked entry.
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
	// kinds (tracked or phantom). Thus, we need to recurse (regardless of their
	// tracked/phantom status), because they could contain phantoms at a lower
	// depth. As we recurse, we'll track whether or not tracked content exists
	// at lower levels. Note that, unlike in the reconcile case, we don't let
	// ancestorContents drive recursion, because that's only necessary to
	// generate deletion changes to the ancestor (whereas here we're not
	// modifying the ancestor).
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

	// Determine how to reify any phantom directories at this level. Any tracked
	// content at lower levels indicates that we should reify phantom
	// directories to tracked directories, as does the presence of a tracked
	// directory in the ancestor. We also take this opportunity to update the
	// directory counts to include this level. In the case that we reify to
	// untracked, we could theoretically lean into the invariant that if both
	// alpha and beta are directory kinds, then they must both be either tracked
	// or phantom, which might allow us to save a few comparisons, but that
	// invariant relies on endpoints behaving correctly, and relying on that
	// would make this code somewhat fragile.
	ancestorIsDirectory := ancestor != nil && ancestor.Kind == EntryKind_Directory
	reifyToTracked := trackedContentExistsAtLowerLevels || ancestorIsDirectory
	if reifyToTracked {
		if alphaIsDirectoryKind {
			alpha.Kind = EntryKind_Directory
			alphaDirectoryCount++
		}
		if betaIsDirectoryKind {
			beta.Kind = EntryKind_Directory
			betaDirectoryCount++
		}
	} else {
		if alphaIsDirectoryKind {
			if alpha.Kind == EntryKind_PhantomDirectory {
				alpha.Kind = EntryKind_Untracked
				alpha.Contents = nil
			} else {
				alphaDirectoryCount++
			}
		}
		if betaIsDirectoryKind {
			if beta.Kind == EntryKind_PhantomDirectory {
				beta.Kind = EntryKind_Untracked
				beta.Contents = nil
			} else {
				betaDirectoryCount++
			}
		}
	}

	// Determine whether or not tracked content exists at or below this level.
	trackedContentExistsAtOrBelowThisLevel := alphaDirectoryCount >= 1 || betaDirectoryCount >= 1

	// Return an indication of whether or not tracked content exists at or below
	// this level, as well as new directory counts.
	return trackedContentExistsAtOrBelowThisLevel, alphaDirectoryCount, betaDirectoryCount
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
