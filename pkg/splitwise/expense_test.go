package splitwise

import (
	"math"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

func TestParseDetails(t *testing.T) {
	tests := []struct {
		name    string
		details string
		want    *ItemizedDetail
	}{
		{
			name:    "empty string",
			details: "",
			want:    nil,
		},
		{
			name: "full detailed breakdown",
			details: "123 - 123.00 (Arran Ubels, test)\n" +
				"234 - 234.00 (Arran Ubels, test)\n" +
				"Tax: Arran Ubels - 0.00, test - 0.00\n" +
				"Tip: Arran Ubels - 0.00, test - 0.00\n",
			want: &ItemizedDetail{
				Notes: "",
				Items: []Item{
					{
						Description: "123",
						Amount:      "123.00",
						SharedWith:  []string{"Arran Ubels", "test"},
					},
					{
						Description: "234",
						Amount:      "234.00",
						SharedWith:  []string{"Arran Ubels", "test"},
					},
				},
				Tax: []PersonAmount{
					{Name: "Arran Ubels", Amount: "0.00"},
					{Name: "test", Amount: "0.00"},
				},
				Tip: []PersonAmount{
					{Name: "Arran Ubels", Amount: "0.00"},
					{Name: "test", Amount: "0.00"},
				},
			},
		},
		{
			name: "items only with compulsory tax and tip",
			details: "Burger - 15.00 (Alice, Bob)\n" +
				"Fries - 5.00 (Alice)\n" +
				"Tax: Alice - 0.00, Bob - 0.00\n" +
				"Tip: Alice - 0.00, Bob - 0.00\n",
			want: &ItemizedDetail{
				Notes: "",
				Items: []Item{
					{
						Description: "Burger",
						Amount:      "15.00",
						SharedWith:  []string{"Alice", "Bob"},
					},
					{
						Description: "Fries",
						Amount:      "5.00",
						SharedWith:  []string{"Alice"},
					},
				},
				Tax: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
				Tip: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
			},
		},
		{
			name: "invalid line skipped",
			details: "Burger - 15.00 (Alice, Bob)\n" +
				"Invalid Line Without Correct Format\n" +
				"Fries - 5.00 (Alice)\n" +
				"Tax: Alice - 0.00, Bob - 0.00\n" +
				"Tip: Alice - 0.00, Bob - 0.00\n",
			want: &ItemizedDetail{
				Notes: "",
				Items: []Item{
					{
						Description: "Burger",
						Amount:      "15.00",
						SharedWith:  []string{"Alice", "Bob"},
					},
					{
						Description: "Fries",
						Amount:      "5.00",
						SharedWith:  []string{"Alice"},
					},
				},
				Tax: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
				Tip: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
			},
		},
		{
			name: "weighted shared with is parsed",
			details: "Taco - 9.00 (Alice*2, Bob)\n" +
				"Tax: Alice - 0.00, Bob - 0.00\n" +
				"Tip: Alice - 0.00, Bob - 0.00\n",
			want: &ItemizedDetail{
				Notes: "",
				Items: []Item{
					{
						Description: "Taco",
						Amount:      "9.00",
						SharedWith:  []string{"Alice", "Alice", "Bob"},
					},
				},
				Tax: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
				Tip: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
			},
		},
		{
			name: "weighted dollar shared with is parsed",
			details: "Taco - 9.00 (Alice $6.00, Bob $3.00)\n" +
				"Tax: Alice - 0.00, Bob - 0.00\n" +
				"Tip: Alice - 0.00, Bob - 0.00\n",
			want: &ItemizedDetail{
				Notes: "",
				Items: []Item{
					{
						Description: "Taco",
						Amount:      "9.00",
						SharedWith:  append(repeatName("Alice", 600), repeatName("Bob", 300)...),
					},
				},
				Tax: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
				Tip: []PersonAmount{
					{Name: "Alice", Amount: "0.00"},
					{Name: "Bob", Amount: "0.00"},
				},
			},
		},
		{
			name: "tax and tip only",
			details: "Tax: Alice - 2.50, Bob - 1.50\n" +
				"Tip: Alice - 1.00, Bob - 1.00\n",
			want: &ItemizedDetail{
				Notes: "",
				Tax: []PersonAmount{
					{Name: "Alice", Amount: "2.50"},
					{Name: "Bob", Amount: "1.50"},
				},
				Tip: []PersonAmount{
					{Name: "Alice", Amount: "1.00"},
					{Name: "Bob", Amount: "1.00"},
				},
			},
		},
		{
			name:    "with notes",
			details: "Some notes here\n\nasdfsdaf\nsdaf\nsd\n\ni1 - 23.00 (Arran Ubels, test)\ni2 - 23.00 (Arran Ubels, test)\nTax: Arran Ubels - 0.00, test - 0.00\nTip: Arran Ubels - 0.00, test - 0.00\n",
			want: &ItemizedDetail{
				Notes: "Some notes here\n\nasdfsdaf\nsdaf\nsd",
				Items: []Item{
					{
						Description: "i1",
						Amount:      "23.00",
						SharedWith:  []string{"Arran Ubels", "test"},
					},
					{
						Description: "i2",
						Amount:      "23.00",
						SharedWith:  []string{"Arran Ubels", "test"},
					},
				},
				Tax: []PersonAmount{
					{Name: "Arran Ubels", Amount: "0.00"},
					{Name: "test", Amount: "0.00"},
				},
				Tip: []PersonAmount{
					{Name: "Arran Ubels", Amount: "0.00"},
					{Name: "test", Amount: "0.00"},
				},
			},
		},
		{
			name:    "notes only (no valid item grammar)",
			details: "Just some notes\nno items here",
			want: &ItemizedDetail{
				Notes: "Just some notes\nno items here",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDetails(tt.details); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDetails() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseAndFormatItemDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantQty     int
		wantDesc    string
		wantFormat  string
	}{
		{
			name:        "plain item defaults to quantity one",
			description: "Burger",
			wantQty:     1,
			wantDesc:    "Burger",
			wantFormat:  "Burger",
		},
		{
			name:        "quantity-prefixed item is parsed",
			description: "3x Taco",
			wantQty:     3,
			wantDesc:    "Taco",
			wantFormat:  "3x Taco",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQty, gotDesc := ParseItemDescription(tt.description)
			if gotQty != tt.wantQty || gotDesc != tt.wantDesc {
				t.Fatalf("ParseItemDescription() = (%d, %q), want (%d, %q)", gotQty, gotDesc, tt.wantQty, tt.wantDesc)
			}
			if got := FormatItemDescription(gotQty, gotDesc); got != tt.wantFormat {
				t.Fatalf("FormatItemDescription() = %q, want %q", got, tt.wantFormat)
			}
		})
	}
}

func TestSerializeDetails(t *testing.T) {
	details := &ItemizedDetail{
		Notes: "Trip notes",
		Items: []Item{
			{Description: "2x Taco", Amount: "9.00", SharedWith: []string{"Alice", "Bob"}},
		},
		Tax: []PersonAmount{
			{Name: "Alice", Amount: "1.00"},
			{Name: "Bob", Amount: "1.00"},
		},
		Tip: []PersonAmount{
			{Name: "Alice", Amount: "0.50"},
			{Name: "Bob", Amount: "0.50"},
		},
	}

	got := SerializeDetails(details)
	want := "Trip notes\n\n2x Taco - 9.00 (Alice, Bob)\nTax: Alice - 1.00, Bob - 1.00\nTip: Alice - 0.50, Bob - 0.50"
	if got != want {
		t.Fatalf("SerializeDetails() = %q, want %q", got, want)
	}
}

func TestSerializeDetailsCompactsWeightedSharedWith(t *testing.T) {
	details := &ItemizedDetail{
		Items: []Item{
			{Description: "12x i2", Amount: "36.00", SharedWith: []string{"Arran Ubels", "Arran Ubels", "test", "test", "test"}},
		},
	}

	got := SerializeDetails(details)
	want := "i2 - 14.40 (Arran Ubels)\ni2 - 21.60 (test)"
	if got != want {
		t.Fatalf("SerializeDetails() = %q, want %q", got, want)
	}

	roundTrip := ParseDetails(got)
	wantRoundTrip := &ItemizedDetail{
		Items: []Item{
			{Description: "i2", Amount: "14.40", SharedWith: []string{"Arran Ubels"}},
			{Description: "i2", Amount: "21.60", SharedWith: []string{"test"}},
		},
	}
	if !reflect.DeepEqual(roundTrip, wantRoundTrip) {
		t.Fatalf("ParseDetails() = %+v, want %+v", roundTrip, wantRoundTrip)
	}
}

func TestSerializeDetailsRoundTripWithHyphenInDescription(t *testing.T) {
	details := &ItemizedDetail{
		Items: []Item{
			{Description: "All Set 25 x 33cm All Purpose Wipes On a Roll - 50 Pack", Amount: "2.50", SharedWith: []string{"Alice", "Bob"}},
		},
	}

	serialized := SerializeDetails(details)
	roundTrip := ParseDetails(serialized)

	if !reflect.DeepEqual(roundTrip, details) {
		t.Fatalf("ParseDetails(SerializeDetails(details)) = %+v, want %+v", roundTrip, details)
	}
}

func repeatName(name string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = name
	}
	return out
}

func TestCalculateOwed(t *testing.T) {
	tests := []struct {
		name     string
		expense  DetailedExpense
		details  ItemizedDetail
		wantOwed []string
		wantNet  []string
	}{
		{
			name: "single item shared 50/50",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{
						User:      User{FirstName: "Alice"},
						PaidShare: "10.00",
					},
					{
						User:      User{FirstName: "Bob"},
						PaidShare: "0.00",
					},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "Burger", Amount: "10.00", SharedWith: []string{"Alice", "Bob"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
			},
			wantOwed: []string{"5.00", "5.00"},
			wantNet:  []string{"5.00", "-5.00"}, // Paid - Owed -> 10-5=5, 0-5=-5
		},
		{
			name: "multiple items unevenly split",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{
						User:      User{FirstName: "Alice"},
						PaidShare: "20.00",
					},
					{
						User:      User{FirstName: "Bob"},
						PaidShare: "0.00",
					},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "Burger", Amount: "15.00", SharedWith: []string{"Alice"}},
					{Description: "Fries", Amount: "5.00", SharedWith: []string{"Bob"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
			},
			wantOwed: []string{"15.00", "5.00"},
			wantNet:  []string{"5.00", "-5.00"}, // 20-15=5, 0-5=-5
		},
		{
			name: "items with tax and tip",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{
						User:      User{FirstName: "Alice"},
						PaidShare: "25.00",
					},
					{
						User:      User{FirstName: "Bob"},
						PaidShare: "0.00",
					},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "Burger", Amount: "20.00", SharedWith: []string{"Alice", "Bob"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "2.00"}, {Name: "Bob", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "3.00"}},
			},
			wantOwed: []string{"12.00", "13.00"},  // Alice: 10 + 2 + 0, Bob: 10 + 0 + 3
			wantNet:  []string{"13.00", "-13.00"}, // 25-12=13, 0-13=-13
		},
		{
			name: "item quantity split by repeated owners",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{
						User:      User{FirstName: "Alice"},
						PaidShare: "9.00",
					},
					{
						User:      User{FirstName: "Bob"},
						PaidShare: "0.00",
					},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "3x Taco", Amount: "9.00", SharedWith: []string{"Alice", "Alice", "Bob"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
			},
			wantOwed: []string{"6.00", "3.00"},
			wantNet:  []string{"3.00", "-3.00"},
		},
		{
			name: "half cent split gives remainder to non payer",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{
						User:      User{FirstName: "Alice"},
						PaidShare: "0.01",
					},
					{
						User:      User{FirstName: "Bob"},
						PaidShare: "0.00",
					},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "Mint", Amount: "0.01", SharedWith: []string{"Alice", "Bob"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
			},
			wantOwed: []string{"0.00", "0.01"},
			wantNet:  []string{"0.01", "-0.01"},
		},
		{
			name: "half cent split gives remainder away from payer regardless of order",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{
						User:      User{FirstName: "Alice"},
						PaidShare: "0.00",
					},
					{
						User:      User{FirstName: "Bob"},
						PaidShare: "0.01",
					},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "Mint", Amount: "0.01", SharedWith: []string{"Alice", "Bob"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}},
			},
			wantOwed: []string{"0.01", "0.00"},
			wantNet:  []string{"-0.01", "0.01"},
		},
		{
			name: "multiple remainder cents are distributed exactly",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{User: User{FirstName: "Alice"}, PaidShare: "0.00"},
					{User: User{FirstName: "Bob"}, PaidShare: "0.00"},
					{User: User{FirstName: "Carol"}, PaidShare: "0.00"},
				},
			},
			details: ItemizedDetail{
				Items: []Item{
					{Description: "Snack", Amount: "1.01", SharedWith: []string{"Alice", "Bob", "Carol"}},
				},
				Tax: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}, {Name: "Carol", Amount: "0.00"}},
				Tip: []PersonAmount{{Name: "Alice", Amount: "0.00"}, {Name: "Bob", Amount: "0.00"}, {Name: "Carol", Amount: "0.00"}},
			},
			wantOwed: nil,
			wantNet:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate updates tt.expense in place
			CalculateOwed(&tt.expense, &tt.details)

			if tt.name == "multiple remainder cents are distributed exactly" {
				var total float64
				var owed []string
				for _, eu := range tt.expense.Users {
					owed = append(owed, eu.OwedShare)
					value, _ := strconv.ParseFloat(eu.OwedShare, 64)
					total += value
				}
				if math.Abs(total-1.01) > 0.0001 {
					t.Fatalf("total owed = %.2f, want 1.01", total)
				}
				sorted := append([]string(nil), owed...)
				sort.Strings(sorted)
				wantSorted := []string{"0.33", "0.34", "0.34"}
				if !reflect.DeepEqual(sorted, wantSorted) {
					t.Fatalf("sorted owed shares = %+v, want %+v", sorted, wantSorted)
				}

				repeatExpense := tt.expense
				for i := range repeatExpense.Users {
					repeatExpense.Users[i].OwedShare = "0.00"
					repeatExpense.Users[i].NetBalance = "0.00"
				}
				CalculateOwed(&repeatExpense, &tt.details)
				for i := range tt.expense.Users {
					if tt.expense.Users[i].OwedShare != repeatExpense.Users[i].OwedShare {
						t.Fatalf("owed share changed between recalculations: first=%q second=%q", tt.expense.Users[i].OwedShare, repeatExpense.Users[i].OwedShare)
					}
				}
				return
			}

			for i, eu := range tt.expense.Users {
				if eu.OwedShare != tt.wantOwed[i] {
					t.Errorf("User %s OwedShare = %s, want %s", eu.User.FirstName, eu.OwedShare, tt.wantOwed[i])
				}
				if eu.NetBalance != tt.wantNet[i] {
					t.Errorf("User %s NetBalance = %s, want %s", eu.User.FirstName, eu.NetBalance, tt.wantNet[i])
				}
			}
		})
	}
}

func TestAutoCorrectPaidShares(t *testing.T) {
	tests := []struct {
		name            string
		expense         DetailedExpense
		previousTotal   float64
		calculatedTotal float64
		wantPaidShares  []string
		wantCost        string
	}{
		{
			name: "no one paying - assign to first payee",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{User: User{FirstName: "Alice"}, PaidShare: "0.00"},
					{User: User{FirstName: "Bob"}, PaidShare: "0.00"},
				},
				Cost: "0.00",
			},
			previousTotal:   0.00,
			calculatedTotal: 25.50,
			wantPaidShares:  []string{"25.50", "0.00"},
			wantCost:        "25.50",
		},
		{
			name: "one person paying - apply delta to their paid share",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{User: User{FirstName: "Alice"}, PaidShare: "0.00"},
					{User: User{FirstName: "Bob"}, PaidShare: "10.00"},
				},
				Cost: "15.00",
			},
			previousTotal:   15.00,
			calculatedTotal: 20.00,
			wantPaidShares:  []string{"0.00", "15.00"},
			wantCost:        "20.00",
		},
		{
			name: "multiple people paying - do not autocorrect (mismatch)",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{User: User{FirstName: "Alice"}, PaidShare: "10.00"},
					{User: User{FirstName: "Bob"}, PaidShare: "5.00"},
				},
				Cost: "15.00",
			},
			previousTotal:   15.00,
			calculatedTotal: 20.00,
			wantPaidShares:  []string{"10.00", "5.00"}, // Unchanged
			wantCost:        "15.00",                   // Unchanged
		},
		{
			name: "same calculated total - preserve manual sole payer edit",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{User: User{FirstName: "Alice"}, PaidShare: "8.00"},
					{User: User{FirstName: "Bob"}, PaidShare: "0.00"},
				},
				Cost: "12.00",
			},
			previousTotal:   12.00,
			calculatedTotal: 12.00,
			wantPaidShares:  []string{"8.00", "0.00"},
			wantCost:        "12.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AutoCorrectPaidShares(&tt.expense, tt.previousTotal, tt.calculatedTotal)

			if tt.expense.Cost != tt.wantCost {
				t.Errorf("Expense Cost = %s, want %s", tt.expense.Cost, tt.wantCost)
			}

			for i, eu := range tt.expense.Users {
				if eu.PaidShare != tt.wantPaidShares[i] {
					t.Errorf("User %s PaidShare = %s, want %s", eu.User.FirstName, eu.PaidShare, tt.wantPaidShares[i])
				}
			}
		})
	}
}
