package importers

import (
	"math"
	"strconv"
	"testing"
)

func TestParseDoorDashEmailTextNewReceipt(t *testing.T) {
	text := `Paid with PayPal
ALDI
Total: A$48.00
Your receipt
123 Fake St, Melbourne VIC 3321, Australia
- For: Arran -

1x  Bakers Life White Lebanese Bread 5 Pack (500 g) (Bakery)
    A$2.89
1x  Beans (375 g) (Produce)
    A$2.49
2x  Broad Oak Farms Free Range Chicken Thigh Fillets Pack (Meat)
    A$21.04
1x  Carrots (1 kg) (Produce)
    A$2.19
1x  Continental Cucumber (Produce)
    A$1.49
1x  Deli Original Pitted Kalamata Olives (350 g) (Pantry)
    A$2.59
1x  Deli Originals Baby Capers (110 g) (Pantry)
    A$2.29
1x  Mars Fun Size Chocolate Bars Share Pack (192 g) (Candy)
    A$5.79
1x  Stonemill Smoked Paprika (40 g) (Pantry)
    A$2.99

Subtotal    A$43.76
Taxes       A$0.00
Bag Fee     A$0.25
Delivery Fee    A$0.00
Service Fee     A$3.99
Tip         A$0.00

Total Charged   A$48.00
Get Order Help`

	parsed, err := ParseDoorDashEmailText(text)
	if err != nil {
		t.Fatalf("ParseDoorDashEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "ALDI" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "ALDI")
	}
	if parsed.Total != "48.00" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "48.00")
	}
	if parsed.CurrencyCode != "AUD" {
		t.Fatalf("CurrencyCode = %q, want %q", parsed.CurrencyCode, "AUD")
	}
	if parsed.SuggestedMode != ImportModeNew {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeNew)
	}
	if len(parsed.Items) != 9 {
		t.Fatalf("len(Items) = %d, want 9", len(parsed.Items))
	}
	if parsed.Items[2].Quantity != 2 || parsed.Items[2].Description != "Broad Oak Farms Free Range Chicken Thigh Fillets Pack (Meat)" || parsed.Items[2].Amount != "21.04" {
		t.Fatalf("third item = %+v", parsed.Items[2])
	}
	if len(parsed.Fees) != 2 {
		t.Fatalf("len(Fees) = %d, want 2", len(parsed.Fees))
	}
	if parsed.Fees[0].Description != "Bag Fee" || parsed.Fees[0].Amount != "0.25" {
		t.Fatalf("first fee = %+v", parsed.Fees[0])
	}
	if parsed.Fees[1].Description != "Service Fee" || parsed.Fees[1].Amount != "3.99" {
		t.Fatalf("second fee = %+v", parsed.Fees[1])
	}
	if parsed.TaxTotal != "0.00" {
		t.Fatalf("TaxTotal = %q, want %q", parsed.TaxTotal, "0.00")
	}
	if parsed.TipTotal != "0.00" {
		t.Fatalf("TipTotal = %q, want %q", parsed.TipTotal, "0.00")
	}
	expectedNotes := `Imported from DoorDash email text
Paid with PayPal
123 Fake St, Melbourne VIC 3321, Australia`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
	assertTotalsMatch(t, parsed)
}

func TestParseDoorDashEmailTextUpdateReceipt(t *testing.T) {
	text := `Paid with PayPal and/or credits

ALDI

Total: A$54.05
Your receipt

123 Fake St, Melbourne VIC 3321, Australia
Items that were adjusted
Out of Stock

1x Bakers Life White Lebanese Bread 5 Pack (500 g)

 A$2.89
Items you ordered

2x Broad Oak Farms Free Range Chicken Thigh Fillets Pack

 A$20.83

A$17.99/kg • Purchased 1.158 kg

1x Deli Original Pitted Kalamata Olives (350 g)

 A$2.59

1x Stonemill Smoked Paprika (40 g)

 A$2.99

1x Neve Neve Marlborough Sauvignon Blanc (750 ml)

 A$8.79

1x Continental Cucumber

 A$1.49

1x Carrots (1 kg)

 A$2.19

1x Deli Originals Baby Capers (110 g)

 A$2.29

1x Beans (375 g)

 A$2.49

1x Mars Fun Size Chocolate Bars Share Pack (192 g)

 A$5.79

Subtotal

A$49.45

Regulatory response fee

A$0.00

Bag Fee

A$0.25

Tax

A$0.00

Delivery fee

A$0.00

Service fee

A$4.35

Dasher tip

A$0.00

Final total charged

A$54.05`

	parsed, err := ParseDoorDashEmailText(text)
	if err != nil {
		t.Fatalf("ParseDoorDashEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "ALDI" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "ALDI")
	}
	if parsed.Total != "54.05" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "54.05")
	}
	if parsed.SuggestedMode != ImportModeUpdate {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeUpdate)
	}
	// Expect 9 items from "Items you ordered" section.
	// The "Out of Stock" item from "Items that were adjusted" should be ignored.
	if len(parsed.Items) != 9 {
		t.Fatalf("len(Items) = %d, want 9", len(parsed.Items))
	}
	if parsed.Items[0].Description != "Broad Oak Farms Free Range Chicken Thigh Fillets Pack" || parsed.Items[0].Quantity != 2 || parsed.Items[0].Amount != "20.83" {
		t.Fatalf("first item = %+v", parsed.Items[0])
	}
	// Explicitly check that the out-of-stock item is NOT in the parsed items
	for _, item := range parsed.Items {
		if item.Description == "Bakers Life White Lebanese Bread 5 Pack (500 g)" {
			t.Fatalf("out-of-stock item should not be imported into ordered items")
		}
	}
	if len(parsed.Fees) != 2 {
		t.Fatalf("len(Fees) = %d, want 2", len(parsed.Fees))
	}
	if parsed.Fees[1].Description != "Service Fee" || parsed.Fees[1].Amount != "4.35" {
		t.Fatalf("second fee = %+v", parsed.Fees[1])
	}
	expectedNotes := `Imported from DoorDash email text
Paid with PayPal and/or credits
123 Fake St, Melbourne VIC 3321, Australia`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
	assertTotalsMatch(t, parsed)
}

func TestParseDoorDashEmailTextSubstitutionReceipt(t *testing.T) {
	text := `Paid with PayPal and/or credits

ALDI

Total: A$44.73
Your receipt

123 Fake St, Melbourne VIC 3321, Australia
Items that were adjusted
Substituted

1x Kensington Pride Mango

 A$2.49

Substituted with:

1x R2E2 Mango

A$2.99
Substituted

1x Lambertz Heart Stars Milk Chocolate Pretzels (500 g)

 A$8.69

Substituted with:

1x Schluckwerder Mozartkugeln Milk & Dark Chocolate (200 g × 10 pk)

A$9.29
Items you ordered

3x Damora Admiral's Quarters Original Water Crackers (125 g)

 A$2.97

1x Strawberries (500 g)

 A$5.99

1x Emporium Selection UK Vintage Cheddar Cheese (200 g)

 A$4.49

1x Moser Roth Finest Milk Chocolate Bars (125 g)

 A$5.79

1x Dairy Fine Chocolate Peanut Clusters (100 g)

 A$3.49

1x R2E2 Mango

 A$2.99

1x Calypso Mango

 A$2.49

Subtotal

A$40.49

Regulatory response fee

A$0.00

Bag Fee

A$0.25

Tax

A$0.00

Delivery fee

A$0.00

Service fee

A$3.99

Dasher tip

A$0.00

Final total charged

A$44.73`

	parsed, err := ParseDoorDashEmailText(text)
	if err != nil {
		t.Fatalf("ParseDoorDashEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "ALDI" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "ALDI")
	}
	if parsed.Total != "44.73" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "44.73")
	}
	if parsed.SuggestedMode != ImportModeUpdate {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeUpdate)
	}

	// Expected items: 7 from "Items you ordered" + 1 new substituted item.
	// One of the "Items you ordered" (R2E2 Mango) will have its quantity incremented.
	if len(parsed.Items) != 8 {
		t.Fatalf("len(Items) = %d, want 8", len(parsed.Items))
	}

	// Helper to find an item by description
	findItem := func(desc string) (ParsedLineItem, bool) {
		for _, item := range parsed.Items {
			if item.Description == desc {
				return item, true
			}
		}
		return ParsedLineItem{}, false
	}

	// Assert R2E2 Mango quantity is 2 (original 1x + substituted 1x)
	r2e2Mango, ok := findItem("R2E2 Mango")
	if !ok {
		t.Fatalf("Expected 'R2E2 Mango' not found in parsed items")
	}
	if r2e2Mango.Quantity != 2 {
		t.Fatalf("R2E2 Mango quantity = %d, want 2", r2e2Mango.Quantity)
	}
	if r2e2Mango.Amount != "2.99" { // Amount should be from the substituted item
		t.Fatalf("R2E2 Mango amount = %q, want %q", r2e2Mango.Amount, "2.99")
	}

	// Assert Schluckwerder Mozartkugeln is present with quantity 1
	mozartkugeln, ok := findItem("Schluckwerder Mozartkugeln Milk & Dark Chocolate (200 g × 10 pk)")
	if !ok {
		t.Fatalf("Expected 'Schluckwerder Mozartkugeln Milk & Dark Chocolate (200 g × 10 pk)' not found in parsed items")
	}
	if mozartkugeln.Quantity != 1 {
		t.Fatalf("Schluckwerder Mozartkugeln quantity = %d, want 1", mozartkugeln.Quantity)
	}
	if mozartkugeln.Amount != "9.29" {
		t.Fatalf("Schluckwerder Mozartkugeln amount = %q, want %q", mozartkugeln.Amount, "9.29")
	}

	// Assert original substituted items are NOT in the final list
	if _, ok := findItem("Kensington Pride Mango"); ok {
		t.Fatalf("original substituted item 'Kensington Pride Mango' should not be imported")
	}
	if _, ok := findItem("Lambertz Heart Stars Milk Chocolate Pretzels (500 g)"); ok {
		t.Fatalf("original substituted item 'Lambertz Heart Stars Milk Chocolate Pretzels (500 g)' should not be imported")
	}

	// Check other items to ensure they are still present and correct (example)
	damoraCrackers, ok := findItem("Damora Admiral's Quarters Original Water Crackers (125 g)")
	if !ok || damoraCrackers.Quantity != 3 || damoraCrackers.Amount != "2.97" {
		t.Fatalf("Damora Crackers incorrect: %+v", damoraCrackers)
	}

	if len(parsed.Fees) != 2 {
		t.Fatalf("len(Fees) = %d, want 2", len(parsed.Fees))
	}
	assertTotalsMatch(t, parsed)
}

func assertTotalsMatch(t *testing.T, parsed *ParsedExpense) {
	t.Helper()
	var calculatedTotal float64
	for _, item := range parsed.Items {
		amount, err := strconv.ParseFloat(item.Amount, 64)
		if err != nil {
			t.Fatalf("Failed to parse item amount %q: %v", item.Amount, err)
		}
		calculatedTotal += amount * float64(item.Quantity) // Multiply by quantity
	}
	for _, fee := range parsed.Fees {
		amount, err := strconv.ParseFloat(fee.Amount, 64)
		if err != nil {
			t.Fatalf("Failed to parse fee amount %q: %v", fee.Amount, err)
		}
		calculatedTotal += amount
	}
	tax, err := strconv.ParseFloat(parsed.TaxTotal, 64)
	if err != nil {
		t.Fatalf("Failed to parse tax total %q: %v", parsed.TaxTotal, err)
	}
	calculatedTotal += tax
	tip, err := strconv.ParseFloat(parsed.TipTotal, 64)
	if err != nil {
		t.Fatalf("Failed to parse tip total %q: %v", parsed.TipTotal, err)
	}
	calculatedTotal += tip

	expectedTotal, err := strconv.ParseFloat(parsed.Total, 64)
	if err != nil {
		t.Fatalf("Failed to parse expected total %q: %v", parsed.Total, err)
	}

	if math.Abs(calculatedTotal-expectedTotal) > 0.001 {
		t.Fatalf("Calculated total (%.2f) does not match expected total (%.2f)", calculatedTotal, expectedTotal)
	}
}
