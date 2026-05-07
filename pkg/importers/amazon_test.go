package importers

import "testing"

func TestParseAmazonEmailTextOrder(t *testing.T) {
	text := `Arriving Wednesday
Example Person - EXAMPLE, VIC
Order # 503-3821217-4635011

View or edit order

MillSO Headset Splitter 3.5mm Jack 3.5mm Female to Dual 3.5mm Male Splitter Adapter for Computer CTIA Headset Mic and Audio Splitter Cable for TRRS Gaming Headset to PC - 8inch/20cm Blue
MillSO Headset Splitter 3.5mm Jac...

Quantity: 1

$799

MillSO 3.5mm Auxiliary Audio Jack to Jack Cable for PC, Notebook, Smartphones, Tablets, MP3 Player and Speakers,24K Gold Plated Male to Male-5M/16.4ft
MillSO 3.5mm Auxiliary Audio Jack...

Quantity: 1

$1599

MillSO Audio Splitter, SapphireBlue&Gold-Plated 8 Feet 3.5mm Male to 2 Male Audio Splitter, TRS Stereo Jack Headphones Adapter Cable for Smartphone, Computer, Mp3, Earphone, Speakers
MillSO Audio Splitter, SapphireBl...

Quantity: 1

$1599

Total $31.98`

	parsed, err := ParseAmazonEmailText(text)
	if err != nil {
		t.Fatalf("ParseAmazonEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "Amazon Order #503-3821217-4635011" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "Amazon Order #503-3821217-4635011")
	}
	if parsed.Total != "31.98" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "31.98")
	}
	if parsed.CurrencyCode != "AUD" {
		t.Fatalf("CurrencyCode = %q, want %q", parsed.CurrencyCode, "AUD")
	}
	if parsed.SuggestedMode != ImportModeNew {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeNew)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "MillSO 3.5mm Auxiliary Audio Jack...", Extra: "MillSO 3.5mm Auxiliary Audio Jack to Jack Cable for PC, Notebook, Smartphones, Tablets, MP3 Player and Speakers,24K Gold Plated Male to Male-5M/16.4ft", Quantity: 1, Amount: "15.99"},
		{Description: "MillSO Audio Splitter, SapphireBl...", Extra: "MillSO Audio Splitter, SapphireBlue&Gold-Plated 8 Feet 3.5mm Male to 2 Male Audio Splitter, TRS Stereo Jack Headphones Adapter Cable for Smartphone, Computer, Mp3, Earphone, Speakers", Quantity: 1, Amount: "15.99"},
		{Description: "MillSO Headset Splitter 3.5mm Jac...", Extra: "MillSO Headset Splitter 3.5mm Jack 3.5mm Female to Dual 3.5mm Male Splitter Adapter for Computer CTIA Headset Mic and Audio Splitter Cable for TRRS Gaming Headset to PC - 8inch/20cm Blue", Quantity: 1, Amount: "7.99"},
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
	expectedNotes := `Imported from Amazon email text
Order # 503-3821217-4635011
Arriving Wednesday
Example Person - EXAMPLE, VIC`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
}

func TestParseAmazonEmailTextFailsOnUnknownItemStructure(t *testing.T) {
	text := `Order # 503-3821217-4635011
Product Title
UNEXPECTED LINE
Quantity: 1
$799
Total $7.99`

	_, err := ParseAmazonEmailText(text)
	if err == nil {
		t.Fatalf("ParseAmazonEmailText() expected error, got nil")
	}
}
