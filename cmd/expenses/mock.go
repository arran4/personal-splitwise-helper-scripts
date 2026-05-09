package main

import (
	"fmt"
	"os"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
	"github.com/arran4/personal-splitwise-helper-scripts/pkg/tui"
)

func runMock() {
	expense := &splitwise.DetailedExpense{
		ID:           123456,
		Description:  "Mock Expense",
		Cost:         "100.00",
		CurrencyCode: "USD",
		Date:         "2023-10-01T12:00:00Z",
		Details: splitwise.SerializeDetails(&splitwise.ItemizedDetail{
			Items: []splitwise.Item{
				{Description: "1x Apples", Amount: "20.00", SharedWith: []string{"Alice", "Bob"}},
				{Description: "2x Bananas", Amount: "30.00", SharedWith: []string{"Alice"}},
				{Description: "1x Coffee", Amount: "20.00", SharedWith: []string{"Bob"}},
				{Description: "3x Oranges", Amount: "18.00", SharedWith: []string{"Alice", "Bob", "Alice"}},
			},
			Tax: []splitwise.PersonAmount{{Name: "Alice", Amount: "1.00"}, {Name: "Bob", Amount: "1.00"}},
			Tip: []splitwise.PersonAmount{{Name: "Alice", Amount: "5.00"}, {Name: "Bob", Amount: "5.00"}},
		}),
		Users: []splitwise.ExpenseUser{
			{
				User:      splitwise.User{ID: 1, FirstName: "Alice"},
				OwedShare: "58.00",
				PaidShare: "100.00",
			},
			{
				User:      splitwise.User{ID: 2, FirstName: "Bob"},
				OwedShare: "42.00",
				PaidShare: "0.00",
			},
		},
		Category: splitwise.Category{
			Name: "Groceries",
		},
	}

	save, payload, err := tui.EditExpense(expense, tui.WithMock(true))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if save {
		fmt.Println("Saved.")
		fmt.Println(string(payload))
	} else {
		fmt.Println("Cancelled.")
	}
}
