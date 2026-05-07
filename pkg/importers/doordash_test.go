package importers

import "testing"

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
	if len(parsed.Items) != 9 {
		t.Fatalf("len(Items) = %d, want 9", len(parsed.Items))
	}
	if parsed.Items[0].Description != "Broad Oak Farms Free Range Chicken Thigh Fillets Pack" || parsed.Items[0].Quantity != 2 || parsed.Items[0].Amount != "20.83" {
		t.Fatalf("first item = %+v", parsed.Items[0])
	}
	if parsed.Items[0].Description == "Bakers Life White Lebanese Bread 5 Pack (500 g)" {
		t.Fatalf("out-of-stock item should not be imported into ordered items")
	}
	if len(parsed.Fees) != 2 {
		t.Fatalf("len(Fees) = %d, want 2", len(parsed.Fees))
	}
	if parsed.Fees[1].Description != "Service Fee" || parsed.Fees[1].Amount != "4.35" {
		t.Fatalf("second fee = %+v", parsed.Fees[1])
	}
}
