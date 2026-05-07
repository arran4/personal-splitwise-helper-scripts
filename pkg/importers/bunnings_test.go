package importers

import "testing"

func TestParseBunningsEmailTextDeliveryOrder(t *testing.T) {
	text := `Thank you for your order - we'll be in touch soon to let you know when your order is ready for delivery.

Your Order Details
Order Date: 17/01/2026
Order Number: W123456789
Flybuys card: 6008********1234

DELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU
W123456789-1
Item Due Qty Total

Cyclone Turbo Blow Torch Kit with Gas
I/N:0729104
2 to 5 business days 1 $114.95

Jumbuck 7kg Lumpwood Charcoal BBQ Fuel
I/N:0132709
2 to 5 business days 1 $19.98

Matador Charcoal Scoop
I/N:0276117
2 to 5 business days 1 $20.96

TOOLS FREE PARCEL
I/N:0598986
0 $0.00

Order Total $155.89
Amount Paid $155.89
Amount Outstanding $0.00
Payment Method
Amount Other $155.89`

	parsed, err := ParseBunningsEmailText(text)
	if err != nil {
		t.Fatalf("ParseBunningsEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "Bunnings" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "Bunnings")
	}
	if parsed.Total != "155.89" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "155.89")
	}
	if parsed.CurrencyCode != "AUD" {
		t.Fatalf("CurrencyCode = %q, want %q", parsed.CurrencyCode, "AUD")
	}
	if parsed.SuggestedMode != ImportModeNew {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeNew)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "Cyclone Turbo Blow Torch Kit with Gas", Extra: "I/N:0729104\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU\nDue: 2 to 5 business days", Quantity: 1, Amount: "114.95"},
		{Description: "Jumbuck 7kg Lumpwood Charcoal BBQ Fuel", Extra: "I/N:0132709\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU\nDue: 2 to 5 business days", Quantity: 1, Amount: "19.98"},
		{Description: "Matador Charcoal Scoop", Extra: "I/N:0276117\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU\nDue: 2 to 5 business days", Quantity: 1, Amount: "20.96"},
		{Description: "TOOLS FREE PARCEL", Extra: "I/N:0598986\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU", Quantity: 0, Amount: "0.00"},
	})
	if len(parsed.Fees) != 0 {
		t.Fatalf("len(Fees) = %d, want 0", len(parsed.Fees))
	}
	if parsed.TaxTotal != "0.00" {
		t.Fatalf("TaxTotal = %q, want %q", parsed.TaxTotal, "0.00")
	}
	if parsed.TipTotal != "0.00" {
		t.Fatalf("TipTotal = %q, want %q", parsed.TipTotal, "0.00")
	}
	expectedNotes := `Imported from Bunnings email text
Order Date: 17/01/2026
Order Number: W123456789
DELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
	assertTotalsMatch(t, parsed)
}

func TestParseBunningsEmailTextSplitCollectionAndDeliveryOrder(t *testing.T) {
	text := `Hi Example Person

Thank you for your order - we'll be in touch soon to let you know when each part of your order is ready for collection/delivery.

Your Order Details
Order Date: 17/01/2026
Order Number: W987654321
Flybuys card: 6008********1234

COLLECTION from: Example Warehouse
W987654321-1
Item Due Qty Total

All Set 25 x 33cm All Purpose Wipes On a Roll - 50 Pack
I/N:4460949
In 2 hours 1 $2.50

Protector Compact Safety Goggles
I/N:5810256
In 2 hours 1 $16.00

Citeco Safety Goggles
I/N:0423453
In 2 hours 1 $11.01

Craftright Dust Masks - 10 Pack
I/N:0375359
In 2 hours 1 $3.15

DELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU
W987654321-2
Item Due Qty Total

Garden Basics 25L All Purpose Potting Mix
I/N:0274301
2 to 5 business days 2 $8.20

OnePass Online Free Delivery
I/N:0368876
0 $0.00

Order Total $40.86
Amount Paid $40.86
Amount Outstanding $0.00
Payment Method
Amount Other $40.86`

	parsed, err := ParseBunningsEmailText(text)
	if err != nil {
		t.Fatalf("ParseBunningsEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "Bunnings" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "Bunnings")
	}
	if parsed.Total != "40.86" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "40.86")
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "All Set 25 x 33cm All Purpose Wipes On a Roll - 50 Pack", Extra: "I/N:4460949\nCOLLECTION from: Example Warehouse\nDue: In 2 hours", Quantity: 1, Amount: "2.50"},
		{Description: "Citeco Safety Goggles", Extra: "I/N:0423453\nCOLLECTION from: Example Warehouse\nDue: In 2 hours", Quantity: 1, Amount: "11.01"},
		{Description: "Craftright Dust Masks - 10 Pack", Extra: "I/N:0375359\nCOLLECTION from: Example Warehouse\nDue: In 2 hours", Quantity: 1, Amount: "3.15"},
		{Description: "Garden Basics 25L All Purpose Potting Mix", Extra: "I/N:0274301\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU\nDue: 2 to 5 business days", Quantity: 2, Amount: "8.20"},
		{Description: "OnePass Online Free Delivery", Extra: "I/N:0368876\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU", Quantity: 0, Amount: "0.00"},
		{Description: "Protector Compact Safety Goggles", Extra: "I/N:5810256\nCOLLECTION from: Example Warehouse\nDue: In 2 hours", Quantity: 1, Amount: "16.00"},
	})
	expectedNotes := `Imported from Bunnings email text
Order Date: 17/01/2026
Order Number: W987654321
COLLECTION from: Example Warehouse
DELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
	assertTotalsMatch(t, parsed)
}

func TestParseBunningsEmailTextFailsOnUnknownItemStructure(t *testing.T) {
	text := `Your Order Details
Order Date: 17/01/2026
Order Number: W123456789

DELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU
W123456789-1
Item Due Qty Total

Cyclone Turbo Blow Torch Kit with Gas
UNEXPECTED LINE
2 to 5 business days 1 $114.95

Order Total $114.95`

	_, err := ParseBunningsEmailText(text)
	if err == nil {
		t.Fatalf("ParseBunningsEmailText() expected error, got nil")
	}
}

func TestParseBunningsEmailTextPartialSnippet(t *testing.T) {
	text := `Order Number: W123456789
DELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU
W123456789-1
Item Due Qty Total

Cyclone Turbo Blow Torch Kit with Gas
I/N:0729104
2 to 5 business days 1 $114.95

Order Total $114.95`

	parsed, err := ParseBunningsEmailText(text)
	if err != nil {
		t.Fatalf("ParseBunningsEmailText() unexpected error: %v", err)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "Cyclone Turbo Blow Torch Kit with Gas", Extra: "I/N:0729104\nDELIVERY to: Unit 1, 123 Example Street EXAMPLE 3000 VIC AU\nDue: 2 to 5 business days", Quantity: 1, Amount: "114.95"},
	})
	if parsed.Total != "114.95" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "114.95")
	}
}

func TestParseBunningsEmailTextTabbedPastedFormat(t *testing.T) {
	text := `Please ensure you bring this order confirmation and your photo ID when you collect from store.

Your Order Details 		Wait for your "Ready to Collect" message before heading in-store to pick up your order
Order Date: 17/01/2026
Order Number: W123456789
Flybuys card: 6008********1234

COLLECTION from: Example Warehouse
W123456789-1
Item 		Due 	Qty 	Total

All Set 25 x 33cm All Purpose Wipes On a Roll - 50 Pack
I/N:4460949
	In 2 hours 	1 	$2.50

Protector Compact Safety Goggles
I/N:5810256
	In 2 hours 	1 	$16.00

DELIVERY to:Unit 1, 123 Example Street EXAMPLE 3000 VIC AU
W123456789-2
Item 		Due 	Qty 	Total

Garden Basics 25L All Purpose Potting Mix
I/N:0274301
	2 to 5 business days 	2 	$8.20

OnePass Online Free Delivery
I/N:0368876
		0 	$0.00
Order Total 	$26.70
Amount Paid 	$26.70
Amount Outstanding 	$0.00
Payment Method
Amount Other 	$26.70`

	parsed, err := ParseBunningsEmailText(text)
	if err != nil {
		t.Fatalf("ParseBunningsEmailText() unexpected error: %v", err)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "All Set 25 x 33cm All Purpose Wipes On a Roll - 50 Pack", Extra: "I/N:4460949\nCOLLECTION from: Example Warehouse\nDue: In 2 hours", Quantity: 1, Amount: "2.50"},
		{Description: "Garden Basics 25L All Purpose Potting Mix", Extra: "I/N:0274301\nDELIVERY to:Unit 1, 123 Example Street EXAMPLE 3000 VIC AU\nDue: 2 to 5 business days", Quantity: 2, Amount: "8.20"},
		{Description: "OnePass Online Free Delivery", Extra: "I/N:0368876\nDELIVERY to:Unit 1, 123 Example Street EXAMPLE 3000 VIC AU", Quantity: 0, Amount: "0.00"},
		{Description: "Protector Compact Safety Goggles", Extra: "I/N:5810256\nCOLLECTION from: Example Warehouse\nDue: In 2 hours", Quantity: 1, Amount: "16.00"},
	})
}
