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
			{Description: "Broad Oak Farms Free Range Chicken Thigh Fillets Pack (Meat)", Extra: "• Skin On\nSpecial Instructions: Trim excess fat", Quantity: 2, Amount: "21.04"},
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
	if expense.Description != "ALDI (28.17 AUD)" {
		t.Fatalf("Description = %q, want %q", expense.Description, "ALDI (28.17 AUD)")
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
	if details.Items[1].Description != "2x Broad Oak Farms Free Range Chicken Thigh Fillets Pack (Meat) | • Skin On | Special Instructions: Trim excess fat" {
		t.Fatalf("second item description = %q", details.Items[1].Description)
	}
	if !reflect.DeepEqual(details.Items[0].SharedWith, []string{"Arran Ubels", "test"}) {
		t.Fatalf("first item shared with = %+v", details.Items[0].SharedWith)
	}
	if expense.Users[0].OwedShare != "14.07" || expense.Users[1].OwedShare != "14.10" {
		t.Fatalf("owed shares = %q / %q, want 14.07 / 14.10", expense.Users[0].OwedShare, expense.Users[1].OwedShare)
	}
}

func TestImportedExpenseMatchScore(t *testing.T) {
	parsed := &importers.ParsedExpense{
		Merchant: "ALDI",
		Total:    "54.05",
		Items:    []importers.ParsedLineItem{{Description: "Item", Quantity: 1, Amount: "1.00"}},
	}
	best := splitwise.Expense{ID: 1, Description: "ALDI", Cost: "54.05"}
	closeExpense := splitwise.Expense{ID: 2, Description: "ALDI Croydon", Cost: "48.00"}
	weak := splitwise.Expense{ID: 3, Description: "Woolworths", Cost: "54.05"}

	bestScore := importedExpenseMatchScore(parsed, best)
	closeScore := importedExpenseMatchScore(parsed, closeExpense)
	weakScore := importedExpenseMatchScore(parsed, weak)

	if bestScore <= closeScore {
		t.Fatalf("best score = %f, close score = %f, want best > close", bestScore, closeScore)
	}
	if weakScore != 0 {
		t.Fatalf("weak score = %f, want 0", weakScore)
	}
}

func TestRankImportedExpenseMatches(t *testing.T) {
	parsed := &importers.ParsedExpense{
		Merchant: "ALDI",
		Total:    "54.05",
		Items:    []importers.ParsedLineItem{{Description: "Item", Quantity: 1, Amount: "1.00"}},
	}
	expenses := []splitwise.Expense{
		{ID: 1, Description: "Woolworths", Cost: "54.05"},
		{ID: 2, Description: "ALDI Croydon", Cost: "48.00"},
		{ID: 3, Description: "ALDI", Cost: "54.05"},
	}

	got := rankImportedExpenseMatches(parsed, expenses)
	if len(got) != 2 {
		t.Fatalf("len(rankImportedExpenseMatches()) = %d, want 2", len(got))
	}
	if got[0].expense.ID != 3 {
		t.Fatalf("first match ID = %d, want 3", got[0].expense.ID)
	}
	if got[1].expense.ID != 2 {
		t.Fatalf("second match ID = %d, want 2", got[1].expense.ID)
	}
}

func TestImportUpdateSelectionLabels(t *testing.T) {
	parsed := &importers.ParsedExpense{
		Merchant: "ALDI",
		Total:    "54.05",
		Items: []importers.ParsedLineItem{
			{Description: "Beans", Quantity: 1, Amount: "2.49"},
			{Description: "Capers", Quantity: 1, Amount: "2.29"},
		},
	}

	if got := importUpdateSelectionTitle(parsed); got != "Select Expense To Update: ALDI 54.05" {
		t.Fatalf("importUpdateSelectionTitle() = %q", got)
	}
	if got := importUpdateSelectionFooter(parsed); got != "Imported update candidate\nMerchant=ALDI Total=54.05 Items=2" {
		t.Fatalf("importUpdateSelectionFooter() = %q", got)
	}
	if got := importMatchSelectionFooter(parsed, 3); got != "Imported update candidate\nMerchant=ALDI Total=54.05 Items=2\nMatched existing expenses: 3\nChoose an existing expense to update or pick New Expense" {
		t.Fatalf("importMatchSelectionFooter() = %q", got)
	}
}

func TestApplyImportedExpensePayerOverride(t *testing.T) {
	expense := &splitwise.DetailedExpense{
		Cost: "42.50",
		Users: []splitwise.ExpenseUser{
			{UserID: 1, PaidShare: "42.50"},
			{UserID: 2, PaidShare: "0.00"},
			{UserID: 3, PaidShare: "0.00"},
		},
	}

	if err := applyImportedExpensePayerOverride(expense, 3); err != nil {
		t.Fatalf("applyImportedExpensePayerOverride() unexpected error: %v", err)
	}
	if expense.Users[0].PaidShare != "0.00" || expense.Users[1].PaidShare != "0.00" || expense.Users[2].PaidShare != "42.50" {
		t.Fatalf("paid shares = %q / %q / %q, want 0.00 / 0.00 / 42.50", expense.Users[0].PaidShare, expense.Users[1].PaidShare, expense.Users[2].PaidShare)
	}
}

func TestApplyImportedExpensePayerOverrideRejectsUnknownUser(t *testing.T) {
	expense := &splitwise.DetailedExpense{
		Cost: "10.00",
		Users: []splitwise.ExpenseUser{
			{UserID: 1, PaidShare: "10.00"},
			{UserID: 2, PaidShare: "0.00"},
		},
	}

	if err := applyImportedExpensePayerOverride(expense, 99); err == nil {
		t.Fatalf("applyImportedExpensePayerOverride() expected error, got nil")
	}
}

func TestResolveFriendPaidUserID(t *testing.T) {
	expense := &splitwise.DetailedExpense{
		Users: []splitwise.ExpenseUser{
			{UserID: 7},
			{UserID: 42},
		},
	}

	got, err := resolveFriendPaidUserID(expense, 7)
	if err != nil {
		t.Fatalf("resolveFriendPaidUserID() unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("resolveFriendPaidUserID() = %d, want 42", got)
	}
}

func TestResolveFriendPaidUserIDRejectsNonPairExpense(t *testing.T) {
	expense := &splitwise.DetailedExpense{
		Users: []splitwise.ExpenseUser{
			{UserID: 7},
			{UserID: 42},
			{UserID: 99},
		},
	}

	if _, err := resolveFriendPaidUserID(expense, 7); err == nil {
		t.Fatalf("resolveFriendPaidUserID() expected error, got nil")
	}
}

func TestImportedExpenseMatchScoreBoostsOrderNumber(t *testing.T) {
	parsed := &importers.ParsedExpense{
		Merchant: "Woolworths Order #284249921",
		Total:    "149.09",
		Items:    []importers.ParsedLineItem{{Description: "Item", Quantity: 1, Amount: "1.00"}},
	}
	withOrderNumber := splitwise.Expense{ID: 1, Description: "Woolworths Order #284249921 (149.09 AUD)", Cost: "149.09"}
	withoutOrderNumber := splitwise.Expense{ID: 2, Description: "Woolworths grocery delivery", Cost: "149.09"}

	withScore := importedExpenseMatchScore(parsed, withOrderNumber)
	withoutScore := importedExpenseMatchScore(parsed, withoutOrderNumber)

	if withScore <= withoutScore {
		t.Fatalf("with order number score = %f, without order number score = %f, want with > without", withScore, withoutScore)
	}
}
