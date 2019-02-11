package sync

// nonDeletionChangesOnly filters a list of changes to only those which are
// non-deletion changes.
func nonDeletionChangesOnly(changes []*Change) []*Change {
	// Create the result.
	// TODO: Should we preallocate here?
	var result []*Change

	// Populate the result.
	for _, c := range changes {
		if c.New != nil {
			result = append(result, c)
		}
	}

	// Done.
	return result
}

// reconciler provides the recursive implementation of reconciliation.
type reconciler struct {
	// synchronizationMode is the synchronization mode to use when determining
	// directionality and conflict resolution behavior.
	synchronizationMode SynchronizationMode
	// ancestorChanges are the changes to the ancestor that are currently being
	// tracked.
	ancestorChanges []*Change
	// alphaChanges are the changes to alpha that are currently being tracked.
	alphaChanges []*Change
	// betaChanges are the changes to beta that are currently being tracked.
	betaChanges []*Change
	// conflicts are the conflicts currently being tracked.
	conflicts []*Conflict
}

// reconcile performs a recursive three-way merge.
func (r *reconciler) reconcile(path string, ancestor, alpha, beta *Entry) {
	// Check if alpha and beta agree on the contents of this path. If so, we can
	// simply recurse.
	if alpha.equalShallow(beta) {
		// If both endpoints agree, grab content lists, because we'll recurse.
		ancestorContents := ancestor.GetContents()
		alphaContents := alpha.GetContents()
		betaContents := beta.GetContents()

		// See if the ancestor also agrees. If it disagrees, record the change
		// for this path and ignore ancestor contents. Since the ancestor is
		// updated with Apply, the Old value will be ignored anyway (since it
		// doesn't need to be transitioned away like on-disk contents do during
		// a transition), so we just leave it nil, rather than set it to the old
		// ancestor contents. Additionally, since we'll be wiping out the old
		// ancestor value at this path, we don't want to recursively add
		// deletion changes for its old contents as well, so we nil them out at
		// this point.
		if !ancestor.equalShallow(alpha) {
			r.ancestorChanges = append(r.ancestorChanges, &Change{
				Path: path,
				New:  alpha.copySlim(),
			})
			ancestorContents = nil
		}

		// Recursively handle contents.
		for name := range nameUnion(ancestorContents, alphaContents, betaContents) {
			r.reconcile(
				pathJoin(path, name),
				ancestorContents[name],
				alphaContents[name],
				betaContents[name],
			)
		}

		// Done.
		return
	}

	// Since there was a disagreement about the contents of this path, we need
	// to disaptch to the appropriate handler.
	switch r.synchronizationMode {
	case SynchronizationMode_SynchronizationModeTwoWaySafe:
		r.handleDisagreementBidirectional(path, ancestor, alpha, beta)
	case SynchronizationMode_SynchronizationModeTwoWayResolved:
		r.handleDisagreementBidirectional(path, ancestor, alpha, beta)
	case SynchronizationMode_SynchronizationModeOneWaySafe:
		r.handleDisagreementUnidirectional(path, ancestor, alpha, beta)
	case SynchronizationMode_SynchronizationModeOneWayReplica:
		r.handleDisagreementUnidirectional(path, ancestor, alpha, beta)
	default:
		panic("unhandled synchronization mode")
	}
}

func (r *reconciler) handleDisagreementBidirectional(path string, ancestor, alpha, beta *Entry) {
	// Since alpha and beta weren't equal at this path, at least one of them
	// must differ from ancestor *at this path*. The other may also differ from
	// the ancestor at this path, a subpath, or not at all. If one side is
	// unmodified, then there is no conflict, and we can simply propagate
	// changes from the other side. This is the standard mechanism for creation,
	// modification, and deletion propagation.
	alphaDelta := diff(path, ancestor, alpha)
	if len(alphaDelta) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  ancestor,
			New:  beta,
		})
		return
	}
	betaDelta := diff(path, ancestor, beta)
	if len(betaDelta) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  ancestor,
			New:  alpha,
		})
		return
	}

	// At this point, we know that both sides have been modified from the
	// ancestor, at least one of them at this path (and the other at either this
	// path or a subpath), and thus a conflict has arisen. We don't know the
	// nature of the changes, and one may be a deletion (though it can't be the
	// case that both are deletions since alpha and beta aren't equal at this
	// path), but if our synchronization mode states that alpha is the
	// unequivocal winner, even in the case of deletions, then we can simply
	// propagate its contents to beta.
	if r.synchronizationMode == SynchronizationMode_SynchronizationModeTwoWayResolved {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha,
		})
		return
	}

	// Next, we try to use our "safe" automatic conflict resolution behavior. If
	// one of the sides contains only deletion changes, then we can safely write
	// over it without losing any new content. This behavior is what enables our
	// form of manual conflict resolution: having the user delete the side they
	// don't want to keep.
	alphaDeltaNonDeletion := nonDeletionChangesOnly(alphaDelta)
	betaDeltaNonDeletion := nonDeletionChangesOnly(betaDelta)
	if len(alphaDeltaNonDeletion) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  alpha,
			New:  beta,
		})
		return
	} else if len(betaDeltaNonDeletion) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha,
		})
		return
	}

	// At this point, both sides have made changes that would cause information
	// to be lost if we were to propgate changes from one side to the other, and
	// we don't have an automatic conflict winner, so we simply record a
	// conflict.
	r.conflicts = append(r.conflicts, &Conflict{
		AlphaChanges: alphaDeltaNonDeletion,
		BetaChanges:  betaDeltaNonDeletion,
	})
}

func (r *reconciler) handleDisagreementUnidirectional(path string, ancestor, alpha, beta *Entry) {
	// If we're performing exact mirroring, then we can simply propagate
	// contents (or lack thereof) from alpha to beta, overwriting any changes
	// that may have occurred on beta.
	if r.synchronizationMode == SynchronizationMode_SynchronizationModeOneWayReplica {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha,
		})
		return
	}

	// At this point, we must be in safe mirroring mode. We thus need to ensure
	// that we don't overwrite any modifications or deletions on beta. There are
	// two cases that we can handle straight away. First, if beta is unmodified,
	// then we know that alpha must be modified, and thus we can propagate over
	// beta. Second, if beta contains only deletion changes, then alpha may or
	// may not be modified, but we should still propagate its contents to either
	// propagate changes or replace the deleted content. Fortunately, both of
	// these cases can be handled with a single check.
	betaDeltaNonDeletion := nonDeletionChangesOnly(diff(path, ancestor, beta))
	if len(betaDeltaNonDeletion) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha,
		})
		return
	}

	// At this point, we know that beta is modified and contains non-deletion
	// changes (either modifications or creations). There is one special case
	// that we can handle here in an automatic and intuitive (from the user
	// perspective) manner: if alpha is nil (i.e. it has no contents due to none
	// having existed or them having been deleted) and it's not the case that
	// both the ancestor and beta are directories (i.e. at least one of them is
	// nil or a non-directory type), then we can simply nil out the ancestor and
	// leave the contents on beta as they are.
	//
	// To understand why this is the only case that we can handle, we have to
	// consider what happens as soon as one of these conditions is not met.
	//
	// If alpha were non-nil, it would mean that there was content on alpha. It
	// wouldn't say anything about whether or not the content was modified (we'd
	// have to do a diff against the ancestor to determine that), but neither
	// case can work. Even if the content is not modified, we still want to
	// repropagate it to enforce mirroring, but we're blocked from doing that by
	// the changes that exist on beta. If the content is modified, then there's
	// an obvious conflict since we couldn't propagate the modification without
	// overwriting the changes on beta. Even if alpha is only subject to
	// deletion changes (i.e. it's a subtree of the ancestor), we still want to
	// maintain the mirroring property of the synchronization, and we can't
	// propagate the deletion without overwriting the contents on beta. You may
	// be asking yourself about the case of alpha and beta both being
	// directories, with alpha having deleted a subset of the tree that doesn't
	// conflict with beta's changes. Well, if both were directories, we wouldn't
	// be here, because we would have simply recursed. At this point, it's
	// guaranteed that one of alpha or beta is not a directory, in which case
	// there's no way that propagation of alpha's (non-nil) contents (modified
	// or not) won't overwrite the changes to beta.
	//
	// The requirement that at least one of ancestor or beta be a (potentially
	// nil) non-directory entry is more subtle and partially heuristically
	// motivated. If both were directories, it would indicate that alpha had
	// also previously been a directory (remember that it can't be now or we
	// would have recursed) and it would not be well-defined which portion of
	// the deletions on alpha should be propagated to the contents of beta. You
	// can't just leave beta as is because that policy would prevent entire
	// directory hierarchies from being deleted, even if only modified partially
	// at a much lower level. Trying to figure out which content on beta should
	// be deleted to "represent" the deletion changes on alpha is neither
	// well-defined nor intuitive. Additionally, at the end of the day, there's
	// no way to delineate the "source" of creation of the directories acting as
	// parents to the modified content (were they "created" on alpha or beta?
	// what if it was due to both-created-same behavior? etc.).
	//
	// Despite the relative complexity of this condition, it still covers a
	// large number of cases. For example, it covers the case that beta creates
	// contents - they are simply not propagated back to alpha. It also covers
	// the case where alpha has deleted something and beta has modified or
	// replaced it - the new beta contents are simply left in place (assuming
	// that they aren't contents at a lower level of a directory hierarchy that
	// alpha has deleted).
	//
	// Finally, since the ancestor is
	// updated with Apply, the Old value will be ignored anyway (since it
	// doesn't need to be transitioned away like on-disk contents do during
	// a transition), so we just set it to nil, rather than the old contents
	// of the ancestor.
	ancestorOrBetaNonDirectory := ancestor == nil ||
		ancestor.Kind != EntryKind_Directory ||
		beta == nil ||
		beta.Kind != EntryKind_Directory
	if alpha == nil && ancestorOrBetaNonDirectory {
		if ancestor != nil {
			// As above, since the ancestor is updated with Apply, the Old value
			// will be ignored anyway (since it doesn't need to be transitioned
			// away like on-disk contents do during a transition), so we just
			// leave it nil, rather than set it to the old ancestor contents.
			r.ancestorChanges = append(r.ancestorChanges, &Change{Path: path})
		}
		return
	}

	// At this point, there's nothing else we can handle using heuristics. We
	// simply have to mark a conflict. Worth noting is that, for alpha, we
	// report all changes, not just non-deletion changes, because even pure
	// deletion changes on alpha's part can be the source of a conflict (unlike
	// in the bidirectional case). For beta, we still report only non-deletion
	// changes, because those are the only changes from which conflict can arise
	// in the unidirectional case. We also don't necessarily know here that
	// alpha is modified - it may not be. In that case, the conflict arises
	// implicitly from the need to mirror alpha's (unchanged) contents to beta,
	// and we still need to ensure that the recorded conflict indicates changes
	// on both endpoints, even if the change on alpha is "synthetic" and
	// represents a change from itself to itself. Fortunately, this "synthetic"
	// change is still an intuitive representation of the source of the
	// conflict.
	alphaDelta := diff(path, ancestor, alpha)
	if len(alphaDelta) == 0 {
		alphaDelta = []*Change{{Path: path, Old: alpha, New: alpha}}
	}
	r.conflicts = append(r.conflicts, &Conflict{
		AlphaChanges: alphaDelta,
		BetaChanges:  betaDeltaNonDeletion,
	})
}

// Reconcile performs a recursive three-way merge and generates a list of
// changes for the ancestor, alpha, and beta, as well as a list of conflicts.
func Reconcile(
	ancestor, alpha, beta *Entry,
	synchronizationMode SynchronizationMode,
) ([]*Change, []*Change, []*Change, []*Conflict) {
	// Create the reconciler.
	r := &reconciler{
		synchronizationMode: synchronizationMode,
	}

	// Perform reconciliation.
	r.reconcile("", ancestor, alpha, beta)

	// Done.
	return r.ancestorChanges, r.alphaChanges, r.betaChanges, r.conflicts
}
