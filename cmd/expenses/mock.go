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
		Users: []splitwise.ExpenseUser{
			{
				User:      splitwise.User{ID: 1, FirstName: "Alice"},
				OwedShare: "50.00",
				PaidShare: "100.00",
			},
			{
				User:      splitwise.User{ID: 2, FirstName: "Bob"},
				OwedShare: "50.00",
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
