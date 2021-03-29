package core

// nonDeletionChangesOnly filters a list of changes to include only non-deletion
// changes (i.e. creations or modifications). The provided slice is modified
// in place to avoid any allocation.
func nonDeletionChangesOnly(changes []*Change) []*Change {
	// Extract the tip of the slice so that we can perform in-place filtering.
	filtered := changes[:0]

	// Perform filtering.
	for _, c := range changes {
		if c.New != nil {
			filtered = append(filtered, c)
		}
	}

	// Done.
	return filtered
}

// reconciler provides the recursive implementation of reconciliation.
type reconciler struct {
	// mode is the synchronization mode to use when determining directionality
	// and conflict resolution behavior.
	mode SynchronizationMode
	// ancestorChanges are the changes to be applied to the ancestor.
	ancestorChanges []*Change
	// alphaChanges are the changes to be applied to alpha.
	alphaChanges []*Change
	// betaChanges are the changes to be applied to beta.
	betaChanges []*Change
	// conflicts are the conflicts between alpha and beta.
	conflicts []*Conflict
}

// reconcile performs recursive reconciliation.
func (r *reconciler) reconcile(path string, ancestor, alpha, beta *Entry) {
	// At the start of this function, we have only one invariant: The ancestor
	// (by definition and enforcement) neither represents nor contains
	// unsynchronizable content. This invariant yields an important corollary:
	// if the recursive diff between the ancestor and another entry is empty or
	// contains only deletion changes, then the other entry neither represents
	// nor contains unsynchronizable content.

	// If either side represents purely problematic content at this path, then
	// there's no point in continuing reconciliation at this path. It's not even
	// worth reporting a conflict because the corresponding problem(s) for this
	// path will already be reported as scan problems and it will be clear to
	// the user why synchronization is not occurring at this path. We also don't
	// attempt any manipulation of the ancestor at this point (even if one side
	// is non-problematic) because we don't know enough about the situation to
	// take any known-correct action.
	if alpha != nil && alpha.Kind == EntryKind_Problematic {
		return
	} else if beta != nil && beta.Kind == EntryKind_Problematic {
		return
	}

	// If both sides are nil or untracked at this path, then we can trivially
	// perform reconciliation by simply niling out the ancestor (if it isn't
	// nil already) because there's nothing to track at this path and no
	// disagreements to resolve.
	alphaNilOrUntracked := alpha == nil || alpha.Kind == EntryKind_Untracked
	betaNilOrUntracked := beta == nil || beta.Kind == EntryKind_Untracked
	if alphaNilOrUntracked && betaNilOrUntracked {
		if ancestor != nil {
			r.ancestorChanges = append(r.ancestorChanges, &Change{Path: path})
		}
		return
	}

	// Check if alpha and beta agree on the contents of this path. If so, then
	// we can simply recurse, because there's no disagreement at this level.
	if alpha.Equal(beta, false) {
		// At this point we know that alpha and beta agree on the content at
		// this path. We also know that neither is problematic at this path and
		// (because we know that they agree on the content of this path and we
		// exclude the both-untracked case) that neither is untracked at this
		// path. Alpha and beta may be directories containing unsynchronizable
		// content at lower levels, but in that case any disagreement will be
		// handled by recursion at those levels.

		// Grab content lists from all entries to enable recursion. We use the
		// accessor functions here because any (or all) of ancestor, alpha, and
		// beta may be nil.
		ancestorContents := ancestor.GetContents()
		alphaContents := alpha.GetContents()
		betaContents := beta.GetContents()

		// See if the ancestor also agrees with alpha and beta at this path. If
		// not, then record a change for this path to make the ancestor agree
		// with the endpoints. This enables the very useful "both modified same"
		// behavior, which supports conflict-free reconciliation of identical
		// creation, modification, and deletion operations performed on both
		// endpoints outside of Mutagen's tracking (e.g. while synchronization
		// sessions are paused or when a synchronization session is created
		// between two existing replicas). Since the ancestor will be updated
		// with Apply (which ignores the Old field of Change), we just leave the
		// Old field nil instead of setting it to the old ancestor contents.
		//
		// It's worth noting that, because of the recursive nature of this
		// algorithm and its depth-first traversal order, we know that any
		// ancestor changes necessary to create parent entries for this entry
		// will already have been recorded and will be performed by Apply first.
		//
		// Finally, since we'll be wiping out the old ancestor value at this
		// path, we don't want to recursively add deletion changes for its old
		// child entries as well, so we nil them out at this point.
		if !ancestor.Equal(alpha, false) {
			r.ancestorChanges = append(r.ancestorChanges, &Change{
				Path: path,
				New:  alpha.Copy(false),
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

	// At this point, our filtering and the lack of shallow equality between
	// alpha and beta add several useful invariants: First, we know that neither
	// alpha nor beta represents problematic content at this path. Second, we
	// know that at most one (but possibly neither) of alpha and beta represents
	// untracked content at this path. Third, we know that at least one of (and
	// possibly both) alpha and beta is (are) non-nil (and thus that one side
	// being nil implies that the other is non-nil). Fourth, we know that alpha
	// and beta disagree on the contents at this path, and that at most one of
	// them (but possibly neither) agrees with the ancestor on the contents at
	// this path. Fifth, we know that at most one (but possibly neither) of
	// alpha and beta is a directory (and thus that one being a directory
	// implies that the other is not). Sixth, as a corollary to the first and
	// fifth invariants, we know that at most one side can contain any
	// problematic content for this path and it must exist at a lower level.
	// Seventh, we know that at least one of the sides represents non-nil
	// synchronizable content at this path.

	// Beyond this point, disagreement handling and conflict resolution depends
	// on the synchronization mode being used. When reasoning about the behavior
	// of these functions, it's important to take into account all of the
	// invariants and corollaries that we've established for this path.
	switch r.mode {
	case SynchronizationMode_SynchronizationModeTwoWaySafe:
		r.handleDisagreementBidirectional(path, ancestor, alpha, beta)
	case SynchronizationMode_SynchronizationModeTwoWayResolved:
		r.handleDisagreementBidirectional(path, ancestor, alpha, beta)
	case SynchronizationMode_SynchronizationModeOneWaySafe:
		r.handleDisagreementOneWaySafe(path, ancestor, alpha, beta)
	case SynchronizationMode_SynchronizationModeOneWayReplica:
		r.handleDisagreementOneWayReplica(path, ancestor, alpha, beta)
	default:
		panic("invalid synchronization mode")
	}
}

// handleDisagreementBidirectional handles content disagreements between alpha
// and beta at a particular path in bidirectional synchronization modes.
func (r *reconciler) handleDisagreementBidirectional(path string, ancestor, alpha, beta *Entry) {
	// At this point, we know that at least one side disagrees with the ancestor
	// at this path, so if one side is entirely unmodified, then we know that
	// the other side must be the one with modifications at this path. In this
	// case, we can simply propagate the synchronizable content from the side
	// that's modified. We also know that the unmodified side can't contain any
	// unsynchronizable content (since it would show up in the diff), so there
	// won't be any problems with removal. This is the classical three-way merge
	// behavior, which handles propagation of most creation, modification, and
	// deletion operations.
	betaDelta := diff(path, ancestor, beta)
	if len(betaDelta) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  ancestor,
			New:  alpha.synchronizable(),
		})
		return
	}
	alphaDelta := diff(path, ancestor, alpha)
	if len(alphaDelta) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  ancestor,
			New:  beta.synchronizable(),
		})
		return
	}

	// At this point, we know that both alpha and beta have been modified, which
	// prevents the classical three-way merge from resolving the disagreement.
	// However, there are still a few heuristics that we can apply to handle a
	// broad range of cases (before yielding a conflict or forced resolution).

	// First, check if both sides have purely deletion changes. If this is the
	// case, then we know that ancestor is a directory, one of alpha or beta is
	// nil, and the other is a pure subtree of the ancestor directory. Ancestor
	// must be a directory because it can't be nil (since we have deletion
	// changes), it can't be unsynchronizable (by definition), and it can't be a
	// synchronizable scalar type because that would require both alpha and beta
	// to be nil (which they aren't) in order to see purely deletion changes on
	// both sides. Since we know that ancestor is a directory, we know that
	// neither alpha nor beta can be a scalar type (since each must be a subtree
	// of ancestor), so they must either both be nil (which is excluded), both
	// be directories (which is also excluded), or be a combination of nil and
	// directory. In this case, one side has completely deleted a directory and
	// the other has partially deleted the directory, so we can simply propagate
	// the full deletion to the side with the partial deletion. Since the side
	// we're wiping out is a pure subtree of the ancestor, we also know that it
	// can't contain any unsynchronizable content, so we won't have any issues
	// with content removal.
	alphaDeltaNonDeletion := nonDeletionChangesOnly(alphaDelta)
	betaDeltaNonDeletion := nonDeletionChangesOnly(betaDelta)
	if len(alphaDeltaNonDeletion) == 0 && len(betaDeltaNonDeletion) == 0 {
		if alpha == nil {
			r.betaChanges = append(r.betaChanges, &Change{
				Path: path,
				Old:  beta,
			})
		} else {
			r.alphaChanges = append(r.alphaChanges, &Change{
				Path: path,
				Old:  alpha,
			})
		}
		return
	}

	// Second, check if just one side has purely deletion changes. In that case,
	// it's reasonable to propagate changes from the other side since we won't
	// lose new content. In this case, we'll filter out the unsynchronizable
	// content from the side with non-deletion changes. We know there can't be
	// any unsynchronizable content on the other side since it has only deletion
	// changes, so we don't need to explicitly check or filter there.
	if len(betaDeltaNonDeletion) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha.synchronizable(),
		})
		return
	} else if len(alphaDeltaNonDeletion) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  alpha,
			New:  beta.synchronizable(),
		})
		return
	}

	// At this point, there are no other heuristics we can apply, so we need to
	// either indicate a conflict or force a resolution, depending on the mode.
	if r.mode == SynchronizationMode_SynchronizationModeTwoWaySafe {
		// In the two-way-safe mode, we simply generate a conflict. We only
		// include non-deletion changes in the conflict since these are the only
		// changes that actually conflict (given our heuristics).
		r.conflicts = append(r.conflicts, &Conflict{
			Root:         path,
			AlphaChanges: alphaDeltaNonDeletion,
			BetaChanges:  betaDeltaNonDeletion,
		})
	} else {
		// In the two-way-resolved mode, alpha always wins over beta, so we want
		// to simply replace the beta contents at this path with those from
		// alpha. However, we won't (and in many cases can't) remove or replace
		// unsynchronizable content, so we need to ensure that beta doesn't
		// contain any unsynchronizable content at or below this path before
		// attempting to propagate contents from alpha. If it does, then we
		// generate a conflict.
		if beta.unsynchronizable() {
			// Compute the conflicting changes on the beta side. In this case,
			// the conflicting changes are only those due to unsynchronizable
			// content. Any other changes (including non-deletion changes)
			// aren't considered conflicting by this mode, which would normally
			// wipe them out.
			betaUnsynchronizableDelta := diff(path, beta.synchronizable(), beta)

			// Record the conflict and bail.
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: alphaDeltaNonDeletion,
				BetaChanges:  betaUnsynchronizableDelta,
			})
			return
		}

		// Generate a change to replace the beta contents at this path with the
		// subset of alpha contents at this path that are synchronizable.
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha.synchronizable(),
		})
	}
}

// handleDisagreementOneWaySafe handles content disagreements between alpha and
// beta at a particular path in the one-way-safe synchronization mode.
func (r *reconciler) handleDisagreementOneWaySafe(path string, ancestor, alpha, beta *Entry) {
	// We're performing safe mirroring, so we need to ensure that we don't
	// overwrite any modifications or deletions on beta. There are two cases
	// that we can handle straight away: First, if beta is unmodified, then we
	// know that alpha must be modified, and thus we can propagate over beta.
	// Second, if beta contains only deletion changes, then alpha may or may not
	// be modified, but we should still propagate alpha's contents to either
	// propagate changes or replace the deleted content. Fortunately, both of
	// these cases can be handled with a single check. It's also important to
	// note that because beta contains no creation changes relative to the
	// ancestor, it also can't contain any unsynchronizable content, so we don't
	// need to check that explicitly, though we do need to filter any
	// unsynchronizable content from alpha before propagating its contents.
	betaDeltaNonDeletion := nonDeletionChangesOnly(diff(path, ancestor, beta))
	if len(betaDeltaNonDeletion) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha.synchronizable(),
		})
		return
	}

	// At this point, we know that beta is modified and contains non-deletion
	// changes (either modifications or creations). There is one special case
	// that we can handle here in an automatic and intuitive manner: if alpha is
	// nil (i.e. it has no contents at this path due to none having existed or
	// them having been deleted) and it's not the case that both the ancestor
	// and beta are directories (i.e. at least one of them is nil or a
	// non-directory type), then we can simply nil out the ancestor (if it isn't
	// nil already) and leave the contents on beta in place. This may seem very
	// specific, but it handles a large number of cases and forms the core of
	// the one-way-safe synchronization logic.
	//
	// To understand why this is the only case that we can handle, we have to
	// consider what happens as soon as one of these conditions is not met.
	//
	// If alpha were non-nil, it would mean that there was content on alpha. It
	// wouldn't say anything about whether or not the content was modified (we'd
	// have to do a diff against the ancestor to determine that), but neither
	// case can work: If alpha is unmodified, we want to repropagate it to
	// enforce mirroring, but we're blocked from doing that by the non-deletion
	// changes that exist on beta, and if alpha is modified, then there's an
	// obvious conflict since we can't propagate the changes from alpha without
	// overwriting the non-deletion changes on beta. Even if alpha is only
	// subject to deletion changes, we still can't propagate those deletions
	// without overwriting the non-deletion changes on beta. You may be asking
	// yourself about the case of alpha and beta both being directories, with
	// alpha having deleted a subset of the tree that doesn't conflict with
	// beta's changes. Well, if both were directories, we wouldn't be here,
	// because we would have simply recursed. At this point, it's guaranteed
	// that one of alpha or beta is not a directory, and as such there's no way
	// that propagation of alpha's (non-nil) contents (modified or not) won't
	// overwrite the changes to beta.
	//
	// The requirement that at least one of ancestor or beta be a (potentially
	// nil) non-directory entry is a bit more subtle and partially heuristically
	// motivated. If both were directories, it would indicate that alpha had
	// also previously been a directory (remember that it can't be now or we
	// would have recursed) and it would not be well-defined which portion of
	// beta should be deleted to reflect the directory deletion on alpha. You
	// can't even use the ancestor to determine the "origin" of contents on beta
	// at that point because it would be ambiguous in cases where alpha and beta
	// happened to agree upon contents at some point in the past.
	//
	// Despite the relative complexity of this condition, it still covers a
	// large number of cases. For example, it covers the case that beta creates
	// contents, in which case those contents are simply not propagated back to
	// alpha. It also covers the case where alpha has deleted something and beta
	// has modified or replaced it, in which case the new beta contents are
	// simply left in place (assuming that they aren't contents at a lower level
	// of a directory hierarchy that alpha has deleted).
	//
	// By ensuring that the ancestor is set to nil in this scenario, we ensure
	// that the contents on beta will be ignored by this same condition on the
	// next synchronization cycle (so long as alpha stays nil).
	untrackBetaContent := alpha == nil &&
		(ancestor == nil || ancestor.Kind != EntryKind_Directory ||
			beta == nil || beta.Kind != EntryKind_Directory)
	if untrackBetaContent {
		if ancestor != nil {
			r.ancestorChanges = append(r.ancestorChanges, &Change{Path: path})
		}
		return
	}

	// At this point, there's nothing else we can handle using heuristics. We
	// simply have to mark a conflict. It's worth noting that we report all
	// changes for alpha, not just non-deletion changes, because even pure
	// deletion changes on alpha's part can be the source of a conflict (unlike
	// in the bidirectional case). It may also be the case that alpha is not
	// modified, in which case the conflict arises implicitly from the desire to
	// mirror alpha's (unchanged) contents to beta. If that's the case, we
	// create a "synthetic" change that indicates alpha has stayed the same.
	// For beta, we still report only non-deletion changes, because those are
	// the only changes from which a conflict can arise in this mode.
	alphaDelta := diff(path, ancestor, alpha)
	if len(alphaDelta) == 0 {
		alphaDelta = []*Change{{Path: path, Old: alpha, New: alpha}}
	}
	r.conflicts = append(r.conflicts, &Conflict{
		Root:         path,
		AlphaChanges: alphaDelta,
		BetaChanges:  betaDeltaNonDeletion,
	})
}

// handleDisagreementOneWayReplica handles content disagreements between alpha
// and beta at a particular path in the one-way-replica synchronization mode.
func (r *reconciler) handleDisagreementOneWayReplica(path string, ancestor, alpha, beta *Entry) {
	// In the one-way-replica mode, we're performing exact mirroring, so we want
	// to simply replace the beta contents at this path with those from alpha.
	// However, we won't (and in many cases can't) remove or replace
	// unsynchronizable content, so we need to ensure that beta doesn't contain
	// any unsynchronizable content at or below this path before attempting to
	// propagate contents from alpha. If it does, then we generate a conflict.
	if beta.unsynchronizable() {
		// Compute the conflicting changes on the alpha side. If there aren't
		// any changes (which may well be the case), then create a "synthetic"
		// change that indicates alpha has stayed the same.
		alphaDelta := diff(path, ancestor, alpha)
		if len(alphaDelta) == 0 {
			alphaDelta = []*Change{{Path: path, Old: alpha, New: alpha}}
		}

		// Compute the conflicting changes on the beta side. In this case, the
		// conflicting changes are only those due to unsynchronizable content.
		// Any other changes (including non-deletion changes) aren't considered
		// conflicting by this mode, which would normally wipe them out.
		betaUnsynchronizableDelta := diff(path, beta.synchronizable(), beta)

		// Record the conflict and bail.
		r.conflicts = append(r.conflicts, &Conflict{
			Root:         path,
			AlphaChanges: alphaDelta,
			BetaChanges:  betaUnsynchronizableDelta,
		})
		return
	}

	// Generate a change to replace the beta contents at this path with the
	// subset of alpha contents at this path that are synchronizable.
	r.betaChanges = append(r.betaChanges, &Change{
		Path: path,
		Old:  beta,
		New:  alpha.synchronizable(),
	})
}

// Reconcile performs a recursive three-way merge and generates a list of
// changes for the ancestor, alpha, and beta, as well as a list of conflicts.
// All of these lists are returned in depth-first but non-deterministic order.
func Reconcile(ancestor, alpha, beta *Entry, mode SynchronizationMode) ([]*Change, []*Change, []*Change, []*Conflict) {
	// Create the reconciler.
	r := &reconciler{mode: mode}

	// Perform reconciliation.
	r.reconcile("", ancestor, alpha, beta)

	// Done.
	return r.ancestorChanges, r.alphaChanges, r.betaChanges, r.conflicts
}
