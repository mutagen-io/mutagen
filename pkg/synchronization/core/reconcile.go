package core

import (
	"github.com/mutagen-io/mutagen/pkg/synchronization/core/fastpath"
)

// extractNonDeletionChanges analyzes a list of changes and generates a new list
// containing only those changes corresponding to non-deletion operations (i.e.
// creations or modifications). The original list is not modified.
func extractNonDeletionChanges(changes []*Change) (filtered []*Change) {
	for _, change := range changes {
		if change.New != nil {
			filtered = append(filtered, change)
		}
	}
	return
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
		// Finally, because we'll be wiping out the old ancestor value at this
		// path, we don't want to recursively add deletion changes for its
		// descendants (if any), and we don't need to because of the
		// aforementioned Apply behavior, so we'll nil out the old ancestor
		// contents to prevent them from driving further traversal.
		if !ancestor.Equal(alpha, false) {
			r.ancestorChanges = append(r.ancestorChanges, &Change{
				Path: path,
				New:  alpha.Copy(EntryCopyBehaviorSlim),
			})
			ancestorContents = nil
		}

		// Compute the prefix to add to content names to compute their paths.
		var contentPathPrefix string
		if len(ancestorContents) > 0 || len(alphaContents) > 0 || len(betaContents) > 0 {
			contentPathPrefix = fastpath.Joinable(path)
		}

		// Recursively handle contents.
		for name := range nameUnion(ancestorContents, alphaContents, betaContents) {
			r.reconcile(
				contentPathPrefix+name,
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
	// Start by extracting the synchronizable portion of each side. These are
	// the entries that we'll use to perform reconciliation. Unsynchronizable
	// content doesn't factor into reconciliation decisions, except to block the
	// propagation of a change to synchronizable content. This filtering doesn't
	// modify any of our invariants. The only cases where it could are those
	// where the equality check in reconcile would yield a different result, but
	// that could only happen if both sides represented purely unsynchronizable
	// content or one was nil while the other was purely unsynchronizable, and
	// we exclude all of those cases with our filtering in reconcile. Thus, the
	// invariants that we've established for alpha and beta still hold for α and
	// β. By performing this filtering, we keep the remainder of reconciliation
	// significantly simpler and more intuitive, while still being able to catch
	// conflicts that would arise due to unsynchronizable content. Historically,
	// before we tracked unsynchronizable content, such content would have
	// manifested as problems during transition operations, so conceptually
	// we're still performing the same reconciliation as we always have, but we
	// now check for blockage due to unsynchronizable content before propagating
	// a change, allowing us to reclassify some transition problems as conflicts
	// and leave on-disk contents in a more coherent state until those conflicts
	// are resolved. In addition to relying on invariant preservation, this
	// strategy also relies on the fact that filtering unsynchronizable content
	// and then performing a diff operation against the ancestor yields either
	// no changes (if the unsynchronizable content was new) or yields a deletion
	// operation (if the unsynchronizable content replaced previously existing
	// synchronizable content), both of which are the behaviors we want for our
	// reconciliation algorithm.
	α := alpha.synchronizable()
	β := beta.synchronizable()

	// At this point, we know that at least one side disagrees with the ancestor
	// at this path, so if one side is entirely unmodified, then we know that
	// the other side must be the one with modifications at this path. In this
	// case, we can simply propagate the synchronizable content from the side
	// that's modified. We also know that the unmodified side can't contain any
	// unsynchronizable content (since it would show up in the diff), so there
	// won't be any problems with removal. This is the classic three-way merge
	// behavior, which propagates most creations, modifications, and deletions.
	αDiff := diff(path, ancestor, α)
	βDiff := diff(path, ancestor, β)
	if len(βDiff) == 0 {
		if betaUnsynchronizable := diff(path, β, beta); len(betaUnsynchronizable) > 0 {
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: αDiff,
				BetaChanges:  betaUnsynchronizable,
			})
		} else {
			r.betaChanges = append(r.betaChanges, &Change{
				Path: path,
				Old:  ancestor,
				New:  α,
			})
		}
		return
	} else if len(αDiff) == 0 {
		if alphaUnsynchronizable := diff(path, α, alpha); len(alphaUnsynchronizable) > 0 {
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: alphaUnsynchronizable,
				BetaChanges:  βDiff,
			})
		} else {
			r.alphaChanges = append(r.alphaChanges, &Change{
				Path: path,
				Old:  ancestor,
				New:  β,
			})
		}
		return
	}

	// At this point, we know that both sides have been modified, so a classic
	// three-way-merge resolution isn't possible. The only other scenarios that
	// we can handle safely (before yielding a conflict or forcing a resolution)
	// are those where one or both sides' changes are purely deletion changes,
	// because those changes don't involve the loss of any new content if we
	// overwrite them with changes from the other side. Thus, we'll start by
	// filtering the diff from each side to identify non-deletion changes.
	αDiffNonDeletion := extractNonDeletionChanges(αDiff)
	βDiffNonDeletion := extractNonDeletionChanges(βDiff)

	// First, check if both sides have purely deletion changes. If this is the
	// case, then we know that ancestor is a directory, one of the sides is nil,
	// and the other side is a non-nil pure subtree of the ancestor directory.
	// Ancestor must be a directory because it can't be nil (since we have
	// deletion changes), it can't be unsynchronizable (by definition), and it
	// can't be a synchronizable scalar type because that would require both
	// sides to be nil (which they can't be at this point) in order to see
	// purely deletion changes in both cases. Since we know that ancestor is a
	// directory, we know that neither side can be a scalar type (since each
	// must be a subtree of ancestor), so they must either both be nil (which is
	// excluded), both be directories (which is also excluded), or be a
	// combination of nil and directory. In this case, one side has completely
	// deleted a directory and the other has partially deleted the directory, so
	// we can simply propagate the full deletion to the side with the partial
	// deletion. Since the side we're wiping out is a pure subtree of the
	// ancestor, we also know that it can't contain any unsynchronizable
	// content, so we won't have any issues with content removal.
	if len(αDiffNonDeletion) == 0 && len(βDiffNonDeletion) == 0 {
		if α == nil {
			if betaUnsynchronizable := diff(path, β, beta); len(betaUnsynchronizable) > 0 {
				r.conflicts = append(r.conflicts, &Conflict{
					Root:         path,
					AlphaChanges: αDiff,
					BetaChanges:  betaUnsynchronizable,
				})
			} else {
				r.betaChanges = append(r.betaChanges, &Change{
					Path: path,
					Old:  β,
				})
			}
		} else {
			if alphaUnsynchronizable := diff(path, α, alpha); len(alphaUnsynchronizable) > 0 {
				r.conflicts = append(r.conflicts, &Conflict{
					Root:         path,
					AlphaChanges: alphaUnsynchronizable,
					BetaChanges:  βDiff,
				})
			} else {
				r.alphaChanges = append(r.alphaChanges, &Change{
					Path: path,
					Old:  α,
				})
			}
		}
		return
	}

	// Second, check if only one side has purely deletion changes. In this case,
	// we know that the other side has creation and/or modification changes (due
	// to the fact that our first heuristic didn't trigger), so we'll want to
	// propagate the content from that side to the one with purely deletion
	// changes. This is what enables our form of manual conflict resolution:
	// deleting the losing side of a conflict.
	//
	// It's worth noting here that if one side has deleted a directory and the
	// other has created or modified content in that directory (excluding the
	// case where the content is purely unsynchronizable), we'll repropagate the
	// entire directory (including its new synchronizable contents) back to the
	// deleted side. This is an intentional behavior. One could alternatively
	// imagine deleting some subset of the creation/modification side (based on
	// what was in the ancestor and therefore deleted on the other side) before
	// propagating the new content, but this rapidly becomes ill-defined (or at
	// least very complex) because you'd also have to preserve the parent
	// directories of the newly created content. Another alternative choice
	// would be to simply indicate a conflict, which would be the behavior of a
	// classic three-way merge algorithm, but there's little practical utility
	// in that, especially when we can perform some sort of resolution action
	// without losing new content. By making the choice to repropagate the whole
	// directory, we're avoiding a conflict and preserving the on-disk "context"
	// for newly created content.
	if len(βDiffNonDeletion) == 0 {
		if betaUnsynchronizable := diff(path, β, beta); len(betaUnsynchronizable) > 0 {
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: αDiffNonDeletion,
				BetaChanges:  betaUnsynchronizable,
			})
		} else {
			r.betaChanges = append(r.betaChanges, &Change{
				Path: path,
				Old:  β,
				New:  α,
			})
		}
		return
	} else if len(αDiffNonDeletion) == 0 {
		if alphaUnsynchronizable := diff(path, α, alpha); len(alphaUnsynchronizable) > 0 {
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: alphaUnsynchronizable,
				BetaChanges:  βDiffNonDeletion,
			})
		} else {
			r.alphaChanges = append(r.alphaChanges, &Change{
				Path: path,
				Old:  α,
				New:  β,
			})
		}
		return
	}

	// At this point, we've seen that both sides have non-deletion chanages, so
	// there are no other heuristics we can apply that don't involve overwriting
	// new content. We need to either indicate a conflict or force a resolution.
	if r.mode == SynchronizationMode_SynchronizationModeTwoWaySafe {
		r.conflicts = append(r.conflicts, &Conflict{
			Root:         path,
			AlphaChanges: αDiffNonDeletion,
			BetaChanges:  βDiffNonDeletion,
		})
	} else {
		if betaUnsynchronizable := diff(path, β, beta); len(betaUnsynchronizable) > 0 {
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: αDiffNonDeletion,
				BetaChanges:  betaUnsynchronizable,
			})
		} else {
			r.betaChanges = append(r.betaChanges, &Change{
				Path: path,
				Old:  β,
				New:  α,
			})
		}
	}
}

// handleDisagreementOneWaySafe handles content disagreements between alpha and
// beta at a particular path in the one-way-safe synchronization mode.
func (r *reconciler) handleDisagreementOneWaySafe(path string, ancestor, alpha, beta *Entry) {
	// Start by extracting the synchronizable portion of beta. The logic behind
	// doing so mirrors that in handleDisagreementBidirectional.
	β := beta.synchronizable()

	// If the synchronizable portion of the beta side is unmodified or contains
	// only deletion changes, then we can simply overwrite it with the content
	// from alpha. The only exception is if beta also contains unsynchronizable
	// content, in which case we indicate a conflict. We could allow this case
	// to be handled by the conflict at the end of this function, but we really
	// want to distinguish between conflicts due to synchronizable content and
	// conflicts due to unsynchronizable content. Doing so allows us to keep a
	// consistent conflict structure with other modes. When determining the
	// alpha changes for the conflict, we're better off using a "synthetic"
	// change of sorts, because alpha may be unchanged or may have a diff that's
	// completely irrelevant to the conflicting unsynchronizable content. In
	// this synthetic change, we don't perform synchronizability filtering for
	// alpha because we don't want alpha to wind up looking nil (in the event
	// that it's purely unsynchronizable content). Specifically, it's better to
	// err on the side of including the unsynchronizable content on alpha,
	// especially since it would have to exist in a directory and any conflict
	// display will probably only show the directory itself as being the
	// conflicting element (since beta is clearly not a directory in that case).
	βDiffNonDeletion := extractNonDeletionChanges(diff(path, ancestor, β))
	if len(βDiffNonDeletion) == 0 {
		if betaUnsynchronizable := diff(path, β, beta); len(betaUnsynchronizable) > 0 {
			r.conflicts = append(r.conflicts, &Conflict{
				Root:         path,
				AlphaChanges: []*Change{{Path: path, Old: ancestor, New: alpha}},
				BetaChanges:  betaUnsynchronizable,
			})
		} else {
			r.betaChanges = append(r.betaChanges, &Change{
				Path: path,
				Old:  beta,
				New:  alpha.synchronizable(),
			})
		}
		return
	}

	// At this point, we know that beta is modified and contains non-deletion
	// changes (either modifications or creations). There is one special case
	// that we can handle here in an automatic and intuitive manner: if alpha is
	// nil (i.e. it has no contents at this path due to none having existed or
	// them having been deleted) or untracked, and it's not the case that both
	// the ancestor and beta are directories (i.e. at least one of them is nil
	// or a non-directory type), then we can simply nil out the ancestor (if it
	// isn't nil already) and leave the contents on beta in place. This may seem
	// very specific, but it handles a large number of cases and forms the core
	// of the one-way-safe synchronization logic.
	//
	// To understand why this is the only case that we can handle, we have to
	// consider what happens as soon as one of these conditions is not met.
	//
	// If alpha were non-nil and not untracked, it would mean that there was
	// synchronizable content on alpha (recall that the purely problematic case
	// has already been excluded). It wouldn't say anything about whether or not
	// the content was modified (we'd have to do a diff against the ancestor to
	// determine that), but neither case can work: If alpha is unmodified, we
	// want to repropagate it to enforce mirroring, but we're blocked from doing
	// that by the non-deletion changes that exist on beta, and if alpha is
	// modified, then there's an obvious conflict since we can't propagate the
	// changes from alpha without overwriting the non-deletion changes on beta.
	// Even if alpha is only subject to partial deletion changes, we still can't
	// propagate the remaining content without overwriting the non-deletion
	// changes on beta.
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
	// next synchronization cycle (so long as alpha stays nil or untracked).
	untrackBetaContent := (alpha == nil || alpha.Kind == EntryKind_Untracked) &&
		(ancestor == nil || ancestor.Kind != EntryKind_Directory ||
			beta == nil || beta.Kind != EntryKind_Directory)
	if untrackBetaContent {
		if ancestor != nil {
			r.ancestorChanges = append(r.ancestorChanges, &Change{Path: path})
		}
		return
	}

	// At this point, there's nothing else we can handle using heuristics, so we
	// simply indicate a conflict. We use a "synethic" change for alpha in this
	// case for the reasons outlined above.
	r.conflicts = append(r.conflicts, &Conflict{
		Root:         path,
		AlphaChanges: []*Change{{Path: path, Old: ancestor, New: alpha}},
		BetaChanges:  βDiffNonDeletion,
	})
}

// handleDisagreementOneWayReplica handles content disagreements between alpha
// and beta at a particular path in the one-way-replica synchronization mode.
func (r *reconciler) handleDisagreementOneWayReplica(path string, ancestor, alpha, beta *Entry) {
	// We're performing exact mirroring, so we simply overwrite whatever exists
	// on beta with the synchronizable contents (or lack thereof) from alpha.
	// The only exception is the case where there's unsynchronizable content on
	// beta (which we can't remove), in which case we indicate a conflict. We
	// use a "synthetic" change for alpha in this case for the reasons outlined
	// in handleDisagreementOneWaySafe.
	if betaUnsynchronizable := diff(path, beta.synchronizable(), beta); len(betaUnsynchronizable) > 0 {
		r.conflicts = append(r.conflicts, &Conflict{
			Root:         path,
			AlphaChanges: []*Change{{Path: path, Old: ancestor, New: alpha}},
			BetaChanges:  betaUnsynchronizable,
		})
	} else {
		r.betaChanges = append(r.betaChanges, &Change{
			Path: path,
			Old:  beta,
			New:  alpha.synchronizable(),
		})
	}
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
