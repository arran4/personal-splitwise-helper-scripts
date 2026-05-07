package importers

import "testing"

func TestParseWoolworthsEmailTextDeliveryOrder(t *testing.T) {
	text := `Thanks for shopping with Woolworths

Hi Example Person

Thank you for your order.

Delivery

Order number: 291029968

Unit 1 123 Example Street, Example 3000

Thursday, 5 February 2026
between 5:00 pm - 6:00 pm

Leave order unattended:

Yes

Notes for the driver:

"leave outside front door"

Your Items

This is not a tax invoice. View in My Orders.
Item description Unit price Qty Price
Woolworths items
Normandie Pate Chicken & Black Peppercorns 5.60 1.00 5.60
Always Fresh Sicilian Olives Pitted 4.80 2.00 9.60
Saxa Table Salt Drum PLAIN 4.50 1.00 4.50
Paper bags: 2.00
Delivery fee: 0.00
Estimated amount to be charged: 21.70
Paid with Credit Card: 21.70`

	parsed, err := ParseWoolworthsEmailText(text)
	if err != nil {
		t.Fatalf("ParseWoolworthsEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "Woolworths Order #291029968" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "Woolworths Order #291029968")
	}
	if parsed.Total != "21.70" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "21.70")
	}
	if parsed.CurrencyCode != "AUD" {
		t.Fatalf("CurrencyCode = %q, want %q", parsed.CurrencyCode, "AUD")
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "Always Fresh Sicilian Olives Pitted", Extra: "", Quantity: 2, Amount: "9.60"},
		{Description: "Normandie Pate Chicken & Black Peppercorns", Extra: "", Quantity: 1, Amount: "5.60"},
		{Description: "Saxa Table Salt Drum PLAIN", Extra: "", Quantity: 1, Amount: "4.50"},
	})
	if len(parsed.Fees) != 1 {
		t.Fatalf("len(Fees) = %d, want 1", len(parsed.Fees))
	}
	if parsed.Fees[0].Description != "Paper Bags" || parsed.Fees[0].Amount != "2.00" {
		t.Fatalf("first fee = %+v", parsed.Fees[0])
	}
	expectedNotes := `Imported from Woolworths email text
Order Number: 291029968
Delivery: Unit 1 123 Example Street, Example 3000
Thursday, 5 February 2026
between 5:00 pm - 6:00 pm`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
	assertTotalsMatch(t, parsed)
}

func TestParseWoolworthsEmailTextFailsOnUnknownItemStructure(t *testing.T) {
	text := `Order number: 291029968
Your Items
Item description Unit price Qty Price
Woolworths items
Normandie Pate Chicken & Black Peppercorns UNEXPECTED
Estimated amount to be charged: 5.60`

	_, err := ParseWoolworthsEmailText(text)
	if err == nil {
		t.Fatalf("ParseWoolworthsEmailText() expected error, got nil")
	}
}

func TestParseWoolworthsEmailTextWeightedItems(t *testing.T) {
	text := `Order number: 290282486
Unit 1 123 Example Street, Example 3000
Saturday, 31 January 2026
between 2:00 pm - 3:00 pm
Your Items
Item description Unit price Qty Price
Woolworths items
Beans Round 6.90 0.25 1.73
Fresh Skin On Barramundi Fillets 36.00 0.56 20.16
Paper bags: 2.00
Estimated amount to be charged: 23.89
Paid with Credit Card: 23.89`

	parsed, err := ParseWoolworthsEmailText(text)
	if err != nil {
		t.Fatalf("ParseWoolworthsEmailText() unexpected error: %v", err)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "Beans Round", Extra: "Unit price: 6.90\nQty: 0.25", Quantity: 1, Amount: "1.73"},
		{Description: "Fresh Skin On Barramundi Fillets", Extra: "Unit price: 36.00\nQty: 0.56", Quantity: 1, Amount: "20.16"},
	})
	if len(parsed.Fees) != 1 || parsed.Fees[0].Amount != "2.00" {
		t.Fatalf("fees = %+v", parsed.Fees)
	}
	assertTotalsMatch(t, parsed)
}

func TestParseWoolworthsEmailTextUpdatedOrder(t *testing.T) {
	text := `Hi Example Person,

Your order has been successfully updated with the changes you submitted recently.
Updated order details

New order number
284249921

Your groceries will arrive on
Saturday, 20 December 2025
between 12:00 pm - 01:00 pm

They'll be delivered to
Unit 1 123 Example Street Example

Your Groceries
Item Description: Unit Price: Quantity: Price:
Lyndale 12 Gippsland's Own Jumbo Cage Free Eggs 6.60 1.00 6.60
Woolworths Cumin Ground Ground 2.00 2.00 3.50
Subtotal: 10.10
Delivery fee 0.00
Paper bags 2.00
Estimated amount to be charged: 12.10
Paid with Credit Card: 12.10`

	parsed, err := ParseWoolworthsEmailText(text)
	if err != nil {
		t.Fatalf("ParseWoolworthsEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "Woolworths Order #284249921" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "Woolworths Order #284249921")
	}
	if parsed.SuggestedMode != ImportModeUpdate {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeUpdate)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "Lyndale 12 Gippsland's Own Jumbo Cage Free Eggs", Extra: "", Quantity: 1, Amount: "6.60"},
		{Description: "Woolworths Cumin Ground Ground", Extra: "", Quantity: 2, Amount: "3.50"},
	})
	expectedNotes := `Imported from Woolworths email text
Order Number: 284249921
Delivery: Unit 1 123 Example Street Example
Saturday, 20 December 2025
between 12:00 pm - 01:00 pm`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
}
