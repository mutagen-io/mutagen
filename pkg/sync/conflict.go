package sync

type Conflict struct {
	AlphaChanges []Change
	BetaChanges  []Change
}
