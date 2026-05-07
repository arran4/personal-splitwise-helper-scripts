package main

import (
	"reflect"
	"testing"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/importers"
	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
)

func TestSplitImportAmountAcrossUsers(t *testing.T) {
	names := []string{"Arran Ubels", "test"}
	got := splitImportAmountAcrossUsers("0.25", names)
	want := []splitwise.PersonAmount{
		{Name: "Arran Ubels", Amount: "0.12"},
		{Name: "test", Amount: "0.13"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitImportAmountAcrossUsers() = %+v, want %+v", got, want)
	}
}

func TestApplyImportedExpense(t *testing.T) {
	lastName := "Ubels"
	expense := &splitwise.DetailedExpense{
		Description:  "Old",
		Cost:         "0.00",
		CurrencyCode: "AUD",
		Date:         "2026-05-07",
		Users: []splitwise.ExpenseUser{
			{
				UserID:    1,
				User:      splitwise.User{ID: 1, FirstName: "Arran", LastName: &lastName},
				PaidShare: "0.00",
				OwedShare: "0.00",
			},
			{
				UserID:    2,
				User:      splitwise.User{ID: 2, FirstName: "test"},
				PaidShare: "0.00",
				OwedShare: "0.00",
			},
		},
	}
	parsed := &importers.ParsedExpense{
		Merchant:     "ALDI",
		CurrencyCode: "AUD",
		Total:        "28.17",
		Notes:        "Imported from DoorDash email text",
		Items: []importers.ParsedLineItem{
			{Description: "Bakers Life White Lebanese Bread 5 Pack (500 g) (Bakery)", Quantity: 1, Amount: "2.89"},
			{Description: "Broad Oak Farms Free Range Chicken Thigh Fillets Pack (Meat)", Quantity: 2, Amount: "21.04"},
		},
		Fees: []importers.ParsedLineItem{
			{Description: "Bag Fee", Quantity: 1, Amount: "0.25"},
			{Description: "Service Fee", Quantity: 1, Amount: "3.99"},
		},
		TaxTotal: "0.00",
		TipTotal: "0.00",
	}

	if err := applyImportedExpense(expense, parsed, false); err != nil {
		t.Fatalf("applyImportedExpense() unexpected error: %v", err)
	}
	if expense.Description != "ALDI" {
		t.Fatalf("Description = %q, want %q", expense.Description, "ALDI")
	}
	if expense.Cost != "28.17" {
		t.Fatalf("Cost = %q, want %q", expense.Cost, "28.17")
	}
	if expense.Users[0].PaidShare != "28.17" {
		t.Fatalf("first user PaidShare = %q, want %q", expense.Users[0].PaidShare, "28.17")
	}
	if expense.Users[1].PaidShare != "0.00" {
		t.Fatalf("second user PaidShare = %q, want %q", expense.Users[1].PaidShare, "0.00")
	}

	details := splitwise.ParseDetails(expense.Details)
	if details == nil {
		t.Fatalf("ParseDetails(expense.Details) returned nil")
	}
	if len(details.Items) != 4 {
		t.Fatalf("len(details.Items) = %d, want 4", len(details.Items))
	}
	if details.Items[1].Description != "2x Broad Oak Farms Free Range Chicken Thigh Fillets Pack (Meat)" {
		t.Fatalf("second item description = %q", details.Items[1].Description)
	}
	if !reflect.DeepEqual(details.Items[0].SharedWith, []string{"Arran Ubels", "test"}) {
		t.Fatalf("first item shared with = %+v", details.Items[0].SharedWith)
	}
	if expense.Users[0].OwedShare != "14.07" || expense.Users[1].OwedShare != "14.10" {
		t.Fatalf("owed shares = %q / %q, want 14.07 / 14.10", expense.Users[0].OwedShare, expense.Users[1].OwedShare)
	}
}
