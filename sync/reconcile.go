package sync

import (
	pathpkg "path"
)

func filterNonDeletion(changes []*Change) []*Change {
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
	resolutions     map[string]ConflictResolution
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
				&Change{path, nil, alpha.copyShallow()},
			)
			ancestorContents = nil
		}

		// Recursively handle contents.
		for n, _ := range iterate(ancestorContents, alphaContents, betaContents) {
			r.reconcile(
				pathpkg.Join(path, n),
				ancestorContents[n],
				alphaContents[n],
				betaContents[n],
			)
		}

		// Done.
		return
	}

	// Alpha and beta weren't equal, so at least one of them must differ from
	// ancestor at this node, and the other may differ at this node or below or
	// not at all. We first check (recursively) if either side is unmodified,
	// and in that case simply propagate the other (which must be the one that
	// differs). If both sides are modified, we check if one side contains only
	// deletions, and if so propagate the other side. This step is an heuristic,
	// and not strictly necessary, but it can solve conflicts automatically
	// without data loss. It must, however, be done AFTER checking for the full
	// delta list, otherwise we wouldn't detect one-sided deletions with the
	// other side unmodified. If both sides contain changes that could be lost,
	// then it's a conflict, so we check for a resolution and if there isn't one
	// we simply record the conflict. When recording a conflict, we only use the
	// non-deletion changes, since this will be displayed to the user and
	// deletion changes don't really matter for conflict resolution.
	alphaDelta := Diff(ancestor, alpha)
	betaDelta := Diff(ancestor, beta)
	alphaDeltaNonDeletion := filterNonDeletion(alphaDelta)
	betaDeltaNonDeletion := filterNonDeletion(betaDelta)
	if len(alphaDelta) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{path, ancestor, beta})
	} else if len(betaDelta) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{path, ancestor, alpha})
	} else if len(alphaDeltaNonDeletion) == 0 {
		r.alphaChanges = append(r.alphaChanges, &Change{path, alpha, beta})
	} else if len(betaDeltaNonDeletion) == 0 {
		r.betaChanges = append(r.betaChanges, &Change{path, beta, alpha})
	} else if resolution, ok := r.resolutions[path]; ok {
		if resolution == ConflictResolution_UseAlpha {
			r.betaChanges = append(r.betaChanges, &Change{path, beta, alpha})
		} else {
			r.alphaChanges = append(r.alphaChanges, &Change{path, alpha, beta})
		}
	} else {
		r.conflicts = append(r.conflicts, &Conflict{
			path,
			alphaDeltaNonDeletion,
			betaDeltaNonDeletion,
		})
	}
}

func Reconcile(
	ancestor, alpha, beta *Entry,
	resolutions map[string]ConflictResolution,
) ([]*Change, []*Change, []*Change, []*Conflict) {
	// Create the reconciler.
	r := &reconciler{resolutions: resolutions}

	// Perform reconciliation.
	r.reconcile("", ancestor, alpha, beta)

	// Done.
	return r.ancestorChanges, r.alphaChanges, r.betaChanges, r.conflicts
}
