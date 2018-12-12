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
	// conflictResolutionMode is the conflict resolution mode to use for
	// handling conflicts.
	conflictResolutionMode ConflictResolutionMode
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
	// Check if alpha and beta agree on the contents of this path.
	if alpha.equalShallow(beta) {
		// If both endpoints agree, grab content lists, because we'll recurse.
		ancestorContents := ancestor.GetContents()
		alphaContents := alpha.GetContents()
		betaContents := beta.GetContents()

		// See if the ancestor also agrees. If it disagrees, record the change
		// for this path and ignore ancestor contents. Since the ancestor is
		// updated with Apply, the Old value will be ignored anyway (since it
		// doesn't need to be transitioned away like on-disk contents do during
		// a transition), so we just set it to nil, rather than the old contents
		// of the ancestor. Since we'll be wiping out the old ancestor value at
		// this path, we don't want to recursively add deletion changes for its
		// old contents as well, so we nil them out at this point.
		if !ancestor.equalShallow(alpha) {
			r.ancestorChanges = append(
				r.ancestorChanges,
				&Change{
					Path: path,
					Old:  nil,
					New:  alpha.CopyShallow(),
				},
			)
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

	// Alpha and beta weren't equal at this path. Thus, at least one of them
	// must differ from ancestor *at this path*. The other may also differ from
	// the ancestor at this path, a subpath, or not at all. If one side is
	// unmodified, then there is no conflict, and we can simply propagate
	// changes from the other side. This is the standard mechanism for change
	// propagation.
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
	// path), but if our conflict resolution mode states that one side is the
	// unequivocal winner, even in the case of deletions, then we can simply
	// propagate that side's contents.
	// NOTE: This is also the point where we'll eventually handle
	// ConflictResolutionMode_ConflictResolutionModeNone, by simply marking a
	// conflict and returning.
	if r.conflictResolutionMode == ConflictResolutionMode_ConflictResolutionModeAlphaWinsAll {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha,
		})
		return
	} else if r.conflictResolutionMode == ConflictResolutionMode_ConflictResolutionModeBetaWinsAll {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  alpha,
			New:  beta,
		})
		return
	}

	// At this point we're dealing with "safe" conflict resolution modes - i.e.
	// those that try to automatically resolve conflicts without losing data
	// (e.g. a modification overriding a deletion). So, before we mark a
	// conflict at this path, check if one side has only deletion changes. If
	// so, then we can propagate the changes from the other side without fear of
	// losing any information. This behavior is what enables our form of manual
	// conflict resolution (having the user delete the side they don't want to
	// keep).
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
	// to be lost if we were to propgate changes from one side to the other. If
	// the conflict resolution mode specifies that one side should win in this
	// case then perform the appropriate resolution, otherwise record a
	// conflict.
	if r.conflictResolutionMode == ConflictResolutionMode_ConflictResolutionModeAlphaWins {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha,
		})
	} else if r.conflictResolutionMode == ConflictResolutionMode_ConflictResolutionModeBetaWins {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  alpha,
			New:  beta,
		})
	} else {
		r.conflicts = append(r.conflicts, &Conflict{
			AlphaChanges: alphaDeltaNonDeletion,
			BetaChanges:  betaDeltaNonDeletion,
		})
	}
}

// Reconcile performs a recursive three-way merge and generates a list of
// changes for the ancestor, alpha, and beta, as well as a list of conflicts.
func Reconcile(
	ancestor,
	alpha,
	beta *Entry,
	conflictResolutionMode ConflictResolutionMode,
) ([]*Change, []*Change, []*Change, []*Conflict) {
	// Create the reconciler.
	r := &reconciler{
		conflictResolutionMode: conflictResolutionMode,
	}

	// Perform reconciliation.
	r.reconcile("", ancestor, alpha, beta)

	// Done.
	return r.ancestorChanges, r.alphaChanges, r.betaChanges, r.conflicts
}
