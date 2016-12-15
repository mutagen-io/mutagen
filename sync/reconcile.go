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
		// for this node and ignore ancestor contents.
		if !alpha.equalShallow(ancestor) {
			r.ancestorChanges = append(
				r.ancestorChanges,
				&Change{
					Path: path,
					Old:  nil,
					New:  alpha.copyShallow(),
				},
			)
			ancestorContents = nil
		}

		// Recursively handle contents.
		iterate3(ancestorContents, alphaContents, betaContents,
			func(name string, a, α, β *Entry) {
				r.reconcile(pathpkg.Join(path, name), a, α, β)
			},
		)

		// Done.
		return
	}

	// Alpha and beta weren't equal at this node. Thus, at least one of them
	// must differ from ancestor *at this node*. The other may also differ from
	// the ancestor at this node, a subnode, or not at all. Start by computing
	// the diff from ancestor to alpha and ancestor to beta. If one side is
	// unmodified, then we can simply propagate changes from the other side.
	alphaDelta := Diff(ancestor, alpha)
	betaDelta := Diff(ancestor, beta)
	if len(alphaDelta) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{
			Path: path,
			Old:  ancestor,
			New:  beta,
		})
		return
	} else if len(betaDelta) == 0 {
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
	// was unmodified. If we only looked at non-deletion changes, we might not
	// detect this because both sides would have no changes or deletion-only
	// changes, and both lists below would be empty, and the winning side would
	// be determined simply by the ordering of the conditional statement below
	// (essentially beta would always win out as it is currently structured).
	//
	// What if both sides have completely deleted this node? Well, that would
	// have passed the equality check at the start of the function and would
	// have been treated as a both-deleted scenario. This, we know at least one
	// side has content at this node.
	//
	// What if both sides are directories and have deleted some subset of the
	// tree below here? Well, that would ALSO have passed the equality check
	// above since nothing has changed at this node, and the function would have
	// simply recursed.
	//
	// Note that, when recording these changes, we use the side we're going to
	// overrule as the "old" value in the change, because that's what it should
	// expext to see on disk, not the ancestor. And since that "old" must be a
	// subtree of ancestor (it contains only deletion changes), it still
	// represents a valid value to return from a transition in the case that the
	// transition fails (no information about previous deletions is lost).
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
	// TODO: We need to come up with more concise conflict representations so
	// that they can be effeciently transmitted to be presented to the user. At
	// the moment they can contain subtrees of effectively unlimited size (that
	// might occur if, e.g., alpha creates a file and beta creates a massive
	// directory hierarchy). I'm thinking we should switch to an enumeration of
	// conflict types that can be paired with the path in question. Even trying
	// to do something like flattening wouldn't help becaues we'd have a path
	// per entry and it'd probably take up more space due to expanding the whole
	// thing out. Also, it's not clear how to efficiently represent these things
	// visually.
	r.conflicts = append(r.conflicts, &Conflict{
		Path:         path,
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
