package splitwise

import (
	"encoding/json"
	"testing"

	"golang.org/x/tools/txtar"
)

func TestEndToEndParsingAndCalculation(t *testing.T) {
	// txtar format file containing input JSON and expected output
	archive := `
-- input.json --
{
  "id": 4447298153,
  "description": "Dinner",
  "details": "Burger - 15.00 (Alice, Bob)\nFries - 5.00 (Alice)\nTax: Alice - 1.00, Bob - 1.00\nTip: Alice - 2.00, Bob - 1.00\n",
  "cost": "25.00",
  "currency_code": "AUD",
  "users": [
	{
	  "user": { "first_name": "Alice" },
	  "paid_share": "25.00",
	  "owed_share": "15.50",
	  "net_balance": "9.50"
	},
	{
	  "user": { "first_name": "Bob" },
	  "paid_share": "0.00",
	  "owed_share": "9.50",
	  "net_balance": "-9.50"
	}
  ]
}
-- expected.json --
{
  "users": [
	{
	  "owed_share": "15.50",
	  "net_balance": "9.50"
	},
	{
	  "owed_share": "9.50",
	  "net_balance": "-9.50"
	}
  ]
}
`

	ar := txtar.Parse([]byte(archive))

	var inputJSON []byte
	var expectedJSON []byte

	for _, f := range ar.Files {
		if f.Name == "input.json" {
			inputJSON = f.Data
		} else if f.Name == "expected.json" {
			expectedJSON = f.Data
		}
	}

	var expense DetailedExpense
	if err := json.Unmarshal(inputJSON, &expense); err != nil {
		t.Fatalf("Failed to unmarshal input JSON: %v", err)
	}

	details := ParseDetails(expense.Details)
	if details == nil {
		t.Fatalf("ParseDetails returned nil")
	}

	// Deliberately corrupt the initial values to prove calculation fixes them
	for i := range expense.Users {
		expense.Users[i].OwedShare = "0.00"
		expense.Users[i].NetBalance = "0.00"
	}

	CalculateOwed(&expense, details)

	// Parse expected struct (partial)
	var expected struct {
		Users []struct {
			OwedShare  string `json:"owed_share"`
			NetBalance string `json:"net_balance"`
		} `json:"users"`
	}
	if err := json.Unmarshal(expectedJSON, &expected); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	for i, eu := range expense.Users {
		wantOwed := expected.Users[i].OwedShare
		wantNet := expected.Users[i].NetBalance
		if eu.OwedShare != wantOwed {
			t.Errorf("User %s OwedShare = %s, want %s", eu.User.FirstName, eu.OwedShare, wantOwed)
		}
		if eu.NetBalance != wantNet {
			t.Errorf("User %s NetBalance = %s, want %s", eu.User.FirstName, eu.NetBalance, wantNet)
		}
	}
}
