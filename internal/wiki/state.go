package wiki

// SyncAction describes the only safe automatic action after fetching.
type SyncAction string

const (
	SyncCurrent     SyncAction = "current"
	SyncAhead       SyncAction = "ahead"
	SyncFastForward SyncAction = "fast-forward"
	SyncDirtyBehind SyncAction = "dirty-behind"
	SyncDiverged    SyncAction = "diverged"
)

// DecideSync deliberately never chooses an operation that could discard or
// hide local work. A clean, behind-only checkout can be fast-forwarded. Local
// changes or divergent commits require the user-facing push reconciliation.
func DecideSync(dirty bool, ahead, behind int) SyncAction {
	if ahead > 0 && behind > 0 {
		return SyncDiverged
	}
	if behind > 0 {
		if dirty {
			return SyncDirtyBehind
		}
		return SyncFastForward
	}
	if ahead > 0 {
		return SyncAhead
	}
	return SyncCurrent
}
