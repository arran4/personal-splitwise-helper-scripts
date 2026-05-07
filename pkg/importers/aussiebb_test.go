package importers

import "testing"

func TestParseAussieBBEmailTextInvoice(t *testing.T) {
	text := `Aussie Broadband Tax Invoice (16/11/2025)
Hi Arran,

Your Aussie Broadband invoice (#53423484) is attached for your records.

Total Due: $85.00

This will be paid by Credit Card on 28-11-2025. Overdue amounts and payment plans still have their original due dates.`

	parsed, err := ParseAussieBBEmailText(text)
	if err != nil {
		t.Fatalf("ParseAussieBBEmailText() unexpected error: %v", err)
	}
	if parsed.Merchant != "Aussie Broadband Tax Invoice #53423484 (16/11/2025)" {
		t.Fatalf("Merchant = %q, want %q", parsed.Merchant, "Aussie Broadband Tax Invoice #53423484 (16/11/2025)")
	}
	if parsed.Total != "85.00" {
		t.Fatalf("Total = %q, want %q", parsed.Total, "85.00")
	}
	if parsed.CurrencyCode != "AUD" {
		t.Fatalf("CurrencyCode = %q, want %q", parsed.CurrencyCode, "AUD")
	}
	if parsed.SuggestedMode != ImportModeNew {
		t.Fatalf("SuggestedMode = %q, want %q", parsed.SuggestedMode, ImportModeNew)
	}
	assertParsedLineItemsEqual(t, parsed.Items, []ParsedLineItem{
		{Description: "Aussie Broadband Tax Invoice #53423484 (16/11/2025)", Quantity: 1, Amount: "85.00"},
	})
	expectedNotes := `Imported from Aussie Broadband email text
Invoice Number: 53423484
Invoice Date: 16/11/2025
Payment: Credit Card on 28-11-2025`
	if parsed.Notes != expectedNotes {
		t.Fatalf("Notes = %q, want %q", parsed.Notes, expectedNotes)
	}
}

func TestParseAussieBBEmailTextFailsOnUnknownStructure(t *testing.T) {
	text := `Aussie Broadband Tax Invoice (16/11/2025)
Your Aussie Broadband invoice (#53423484) is attached for your records.
This will be paid by Credit Card on 28-11-2025.`

	_, err := ParseAussieBBEmailText(text)
	if err == nil {
		t.Fatalf("ParseAussieBBEmailText() expected error, got nil")
	}
}
