package sync

type Conflict struct {
	Path         string
	AlphaChanges []Change
	BetaChanges  []Change
}
