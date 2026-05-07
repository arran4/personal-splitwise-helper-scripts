package tui

import "testing"

func TestScoreSelectionMatch(t *testing.T) {
	tests := []struct {
		name      string
		candidate string
		query     string
		wantScore int
		wantMatch bool
	}{
		{
			name:      "contiguous match after prefix",
			candidate: "123asdf",
			query:     "asd",
			wantScore: 3,
			wantMatch: true,
		},
		{
			name:      "gapped match accumulates intervening characters",
			candidate: "1a2s34d",
			query:     "asd",
			wantScore: 4,
			wantMatch: true,
		},
		{
			name:      "out of order does not match",
			candidate: "123asdf",
			query:     "asg",
			wantScore: 0,
			wantMatch: false,
		},
		{
			name:      "lowercase query does not penalize uppercase candidate",
			candidate: "ASD",
			query:     "asd",
			wantScore: 0,
			wantMatch: true,
		},
		{
			name:      "uppercase query penalizes lowercase candidate",
			candidate: "asd",
			query:     "Asd",
			wantScore: 1,
			wantMatch: true,
		},
		{
			name:      "uppercase query matches uppercase candidate without penalty",
			candidate: "Asd",
			query:     "Asd",
			wantScore: 0,
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScore, gotMatch := scoreSelectionMatch(tt.candidate, tt.query)
			if gotScore != tt.wantScore || gotMatch != tt.wantMatch {
				t.Fatalf("scoreSelectionMatch(%q, %q) = (%d, %v), want (%d, %v)", tt.candidate, tt.query, gotScore, gotMatch, tt.wantScore, tt.wantMatch)
			}
		})
	}
}

func TestFilterSelectionOptionsSortsByScore(t *testing.T) {
	options := []SelectionOption{
		{Label: "1a2s34d"},
		{Label: "123asdf"},
		{Label: "zzz"},
	}

	got := filterSelectionOptions(options, "asd")
	if len(got) != 2 {
		t.Fatalf("len(filterSelectionOptions()) = %d, want 2", len(got))
	}
	if got[0].label != "123asdf" || got[0].score != 3 {
		t.Fatalf("first result = %+v, want label %q score 3", got[0], "123asdf")
	}
	if got[1].label != "1a2s34d" || got[1].score != 4 {
		t.Fatalf("second result = %+v, want label %q score 4", got[1], "1a2s34d")
	}
}

func TestFilterTableSelectionOptionsUsesJoinedCells(t *testing.T) {
	options := []TableSelectionOption{
		{Cells: []string{"2026-05-07", "Dinner", "Arran Ubels", "12.00 AUD"}},
		{Cells: []string{"2026-05-06", "Coffee", "test", "4.00 AUD"}},
	}

	got := filterTableSelectionOptions(options, "Arr")
	if len(got) != 1 {
		t.Fatalf("len(filterTableSelectionOptions()) = %d, want 1", len(got))
	}
	if got[0].cells[2] != "Arran Ubels" {
		t.Fatalf("matched row = %+v, want recipient %q", got[0].cells, "Arran Ubels")
	}
}
