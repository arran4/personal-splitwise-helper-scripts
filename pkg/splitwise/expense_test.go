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
			name: "items only",
			details: "Burger - 15.00 (Alice, Bob)\n" +
				"Fries - 5.00 (Alice)\n",
			want: &ItemizedDetail{
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
			},
		},
		{
			name: "invalid line skipped",
			details: "Burger - 15.00 (Alice, Bob)\n" +
				"Invalid Line Without Correct Format\n" +
				"Fries - 5.00 (Alice)\n",
			want: &ItemizedDetail{
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
			},
		},
		{
			name:    "tax only",
			details: "Tax: Alice - 2.50, Bob - 1.50\n",
			want: &ItemizedDetail{
				Tax: []PersonAmount{
					{Name: "Alice", Amount: "2.50"},
					{Name: "Bob", Amount: "1.50"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDetails(tt.details); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}
