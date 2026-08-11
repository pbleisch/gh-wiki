package wiki

import "testing"

func TestDecideSync(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		dirty         bool
		ahead, behind int
		want          SyncAction
	}{
		{name: "clean current", want: SyncCurrent},
		{name: "dirty current", dirty: true, want: SyncCurrent},
		{name: "ahead", ahead: 2, want: SyncAhead},
		{name: "dirty ahead", dirty: true, ahead: 2, want: SyncAhead},
		{name: "clean behind", behind: 3, want: SyncFastForward},
		{name: "dirty behind", dirty: true, behind: 3, want: SyncDirtyBehind},
		{name: "diverged", ahead: 2, behind: 3, want: SyncDiverged},
		{name: "dirty diverged", dirty: true, ahead: 2, behind: 3, want: SyncDiverged},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := DecideSync(test.dirty, test.ahead, test.behind); got != test.want {
				t.Errorf("DecideSync(%v, %d, %d) = %q, want %q", test.dirty, test.ahead, test.behind, got, test.want)
			}
		})
	}
}
