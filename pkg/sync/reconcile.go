package sync

import (
	pathpkg "path"
)

func nonDeletionChangesOnly(changes []*Change) []*Change {
	var result []*Change
	for _, c := range changes {
		if c.New != nil {
			result = append(result, c)
		}
	}
	return result
}

type reconciler struct {
	ancestorChanges []*Change
	alphaChanges    []*Change
	betaChanges     []*Change
	conflicts       []*Conflict
}

func (r *reconciler) reconcile(path string, ancestor, alpha, beta *Entry) {
	// Check if alpha and beta agree on the contents of this node.
	if alpha.equalShallow(beta) {
		// If both endpoints agree, grab content lists, because we'll recurse.
		ancestorContents := ancestor.GetContents()
		alphaContents := alpha.GetContents()
		betaContents := beta.GetContents()

		// See if the ancestor also agrees. If it disagrees, record the change
		// for this node and ignore ancestor contents. We ignore the contents so
		// that we don't add deletion changes for old subnodes.
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
				pathpkg.Join(path, name),
				ancestorContents[name],
				alphaContents[name],
				betaContents[name],
			)
		}

		// Done.
		return
	}

	// Alpha and beta weren't equal at this node. Thus, at least one of them
	// must differ from ancestor *at this node*. The other may also differ from
	// the ancestor at this node, a subnode, or not at all. If one side is
	// unmodified, then we can simply propagate changes from the other side.
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

	// It appears that both sides have been modified. Before we mark a conflict,
	// check if one side has only deletion changes. If so, we can propagate the
	// changes from the other side without fear of losing any information. This
	// is essentially the only form of automated conflict resolution that we can
	// do. In some sense, it is a heuristic designed to avoid conflicts in very
	// common cases, but more importantly, it is necessary to enable our form of
	// manual conflict resolution: having the user delete the side they don't
	// want to keep.
	//
	// Now, you're probably asking yourself a few questions here:
	//
	// Why didn't we simply make this check first? Why do we need to check the
	// full diffs above? Well, imagine that one side had deleted and the other
	// was unmodified. If we only looked at non-deletion changes, we would not
	// detect this because both sides would have no changes or deletion-only
	// changes, and both lists below would be empty, and the winning side would
	// be determined simply by the ordering of the conditional statement below
	// (essentially beta would always win out as it is currently structured).
	//
	// What if both sides have completely deleted this node? Well, that would
	// have passed the equality check at the start of the function and would
	// have been treated as a both-deleted scenario. Thus, we know at least one
	// side has content at this node.
	//
	// What if both sides are directories and have only deleted some subset of
	// the tree below here? Well, that would ALSO have passed the equality check
	// above since nothing has changed at this node, and the function would have
	// simply recursed.
	//
	// Note that, when recording these changes, we use the side we're going to
	// overrule as the "old" value in the change, because that's what it should
	// expect to see on disk, not the ancestor. And since that "old" must be a
	// subtree of ancestor (it contains only deletion changes), it still
	// represents a valid value to return from a transition in the case that the
	// transition fails, and, as a nice side-effect in that case, no information
	// about the deletions that have happened on that side is lost.
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
	// to be lost if we were to propgate changes from one side to the other, so
	// we need to record a conflict. We only record non-deletion changes because
	// those are the only ones that create conflict.
	r.conflicts = append(r.conflicts, &Conflict{
		AlphaChanges: alphaDeltaNonDeletion,
		BetaChanges:  betaDeltaNonDeletion,
	})
}

func Reconcile(ancestor, alpha, beta *Entry) ([]*Change, []*Change, []*Change, []*Conflict) {
	// Create the reconciler.
	r := &reconciler{}

	// Perform reconciliation.
	r.reconcile("", ancestor, alpha, beta)

	// Done.
	return r.ancestorChanges, r.alphaChanges, r.betaChanges, r.conflicts
}
