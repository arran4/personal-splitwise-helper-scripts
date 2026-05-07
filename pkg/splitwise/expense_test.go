package splitwise

import (
	"reflect"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate updates tt.expense in place
			CalculateOwed(&tt.expense, &tt.details)

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
			calculatedTotal: 25.50,
			wantPaidShares:  []string{"25.50", "0.00"},
			wantCost:        "25.50",
		},
		{
			name: "one person paying - update their total",
			expense: DetailedExpense{
				Users: []ExpenseUser{
					{User: User{FirstName: "Alice"}, PaidShare: "0.00"},
					{User: User{FirstName: "Bob"}, PaidShare: "15.00"},
				},
				Cost: "15.00",
			},
			calculatedTotal: 20.00,
			wantPaidShares:  []string{"0.00", "20.00"},
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
			calculatedTotal: 20.00,
			wantPaidShares:  []string{"10.00", "5.00"}, // Unchanged
			wantCost:        "15.00",                   // Unchanged
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AutoCorrectPaidShares(&tt.expense, tt.calculatedTotal)

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
