package main

import "testing"

func TestParsePagesFlag(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    pagePlan
		wantErr bool
	}{
		{
			name: "default single page",
			raw:  "",
			want: pagePlan{startPage: 0, pageCount: 1, fetchAll: false},
		},
		{
			name: "all pages",
			raw:  "all",
			want: pagePlan{startPage: 1, pageCount: 0, fetchAll: true},
		},
		{
			name: "single explicit page",
			raw:  "3",
			want: pagePlan{startPage: 3, pageCount: 1, fetchAll: false},
		},
		{
			name: "bounded page range",
			raw:  "3-4",
			want: pagePlan{startPage: 3, pageCount: 2, fetchAll: false},
		},
		{
			name: "open ended page range",
			raw:  "3-",
			want: pagePlan{startPage: 3, pageCount: 0, fetchAll: true},
		},
		{
			name:    "invalid descending range",
			raw:     "4-3",
			wantErr: true,
		},
		{
			name:    "invalid zero page",
			raw:     "0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePagesFlag(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePagesFlag(%q) expected error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePagesFlag(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("parsePagesFlag(%q) = %+v, want %+v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestNewRecentExpensePageCursor(t *testing.T) {
	tests := []struct {
		name string
		opts expenseListOptions
		want recentExpensePageCursor
	}{
		{
			name: "default chooser starts at offset and can keep loading",
			opts: expenseListOptions{limit: 20, offset: 40},
			want: recentExpensePageCursor{
				limit:          20,
				nextOffset:     40,
				remainingPages: 0,
				fetchAll:       true,
			},
		},
		{
			name: "bounded page range starts at requested page",
			opts: expenseListOptions{limit: 20, pages: "3-4"},
			want: recentExpensePageCursor{
				limit:          20,
				nextOffset:     40,
				remainingPages: 2,
				fetchAll:       false,
			},
		},
		{
			name: "all pages starts at page one",
			opts: expenseListOptions{limit: 10, pages: "all"},
			want: recentExpensePageCursor{
				limit:          10,
				nextOffset:     0,
				remainingPages: 0,
				fetchAll:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newRecentExpensePageCursor(tt.opts)
			if err != nil {
				t.Fatalf("newRecentExpensePageCursor(%+v) unexpected error: %v", tt.opts, err)
			}
			if got != tt.want {
				t.Fatalf("newRecentExpensePageCursor(%+v) = %+v, want %+v", tt.opts, got, tt.want)
			}
		})
	}
}

func TestRecentExpensePageCursorConsumePage(t *testing.T) {
	t.Run("bounded cursor stops after requested pages", func(t *testing.T) {
		cursor := recentExpensePageCursor{limit: 20, nextOffset: 40, remainingPages: 2, fetchAll: false}
		if !cursor.consumePage(20) {
			t.Fatalf("first full page should still allow more")
		}
		if cursor.nextOffset != 60 || cursor.remainingPages != 1 {
			t.Fatalf("after first page cursor = %+v", cursor)
		}
		if cursor.consumePage(20) {
			t.Fatalf("second full page should exhaust bounded cursor")
		}
		if cursor.nextOffset != 80 || cursor.remainingPages != 0 {
			t.Fatalf("after second page cursor = %+v", cursor)
		}
	})

	t.Run("short page ends unbounded cursor", func(t *testing.T) {
		cursor := recentExpensePageCursor{limit: 20, nextOffset: 0, remainingPages: 0, fetchAll: true}
		if cursor.consumePage(7) {
			t.Fatalf("short page should stop pagination")
		}
		if cursor.fetchAll || cursor.remainingPages != 0 || cursor.nextOffset != 20 {
			t.Fatalf("after short page cursor = %+v", cursor)
		}
	})
}
