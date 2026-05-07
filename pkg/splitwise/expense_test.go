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
