package importers

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	aussieBBInvoiceDatePattern   = regexp.MustCompile(`\((\d{2}/\d{2}/\d{4})\)`)
	aussieBBInvoiceNumberPattern = regexp.MustCompile(`invoice\s*\(#?([0-9]+)\)`)
	aussieBBTotalPattern         = regexp.MustCompile(`(?i)^total due:\s*\$([0-9]+(?:\.[0-9]{2})?)$`)
	aussieBBPaymentPattern       = regexp.MustCompile(`(?i)^this will be paid by (.+?) on (\d{2}-\d{2}-\d{4})\.`)
)

func ParseAussieBBEmailText(text string) (*ParsedExpense, error) {
	lines := normalizeImportLines(text)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no Aussie Broadband email text found")
	}

	invoiceDate := parseAussieBBInvoiceDate(lines)
	if invoiceDate == "" {
		return nil, fmt.Errorf("could not determine Aussie Broadband invoice date from email text")
	}

	invoiceNumber := parseAussieBBInvoiceNumber(lines)
	if invoiceNumber == "" {
		return nil, fmt.Errorf("could not determine Aussie Broadband invoice number from email text")
	}

	total := parseAussieBBTotal(lines)
	if total == "" {
		return nil, fmt.Errorf("could not determine Aussie Broadband total due from email text")
	}

	merchant := buildAussieBBMerchant(invoiceNumber, invoiceDate)
	return &ParsedExpense{
		Merchant:      merchant,
		CurrencyCode:  "AUD",
		Total:         total,
		Notes:         buildAussieBBNotes(lines, invoiceNumber, invoiceDate),
		Items:         []ParsedLineItem{{Description: merchant, Quantity: 1, Amount: total}},
		TaxTotal:      "0.00",
		TipTotal:      "0.00",
		SuggestedMode: ImportModeNew,
	}, nil
}

func parseAussieBBInvoiceDate(lines []string) string {
	for _, line := range lines {
		if !strings.Contains(strings.ToLower(line), "aussie broadband") || !strings.Contains(strings.ToLower(line), "tax invoice") {
			continue
		}
		if matches := aussieBBInvoiceDatePattern.FindStringSubmatch(line); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func parseAussieBBInvoiceNumber(lines []string) string {
	for _, line := range lines {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "aussie broadband invoice") {
			continue
		}
		if matches := aussieBBInvoiceNumberPattern.FindStringSubmatch(lower); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func parseAussieBBTotal(lines []string) string {
	for _, line := range lines {
		if matches := aussieBBTotalPattern.FindStringSubmatch(strings.TrimSpace(line)); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

func buildAussieBBMerchant(invoiceNumber, invoiceDate string) string {
	return fmt.Sprintf("Aussie Broadband Tax Invoice #%s (%s)", invoiceNumber, invoiceDate)
}

func buildAussieBBNotes(lines []string, invoiceNumber, invoiceDate string) string {
	notes := []string{
		"Imported from Aussie Broadband email text",
		fmt.Sprintf("Invoice Number: %s", invoiceNumber),
		fmt.Sprintf("Invoice Date: %s", invoiceDate),
	}
	if method, paymentDate := parseAussieBBPayment(lines); method != "" || paymentDate != "" {
		if method != "" && paymentDate != "" {
			notes = append(notes, fmt.Sprintf("Payment: %s on %s", method, paymentDate))
		} else if method != "" {
			notes = append(notes, "Payment: "+method)
		} else {
			notes = append(notes, "Payment Date: "+paymentDate)
		}
	}
	return strings.Join(notes, "\n")
}

func parseAussieBBPayment(lines []string) (string, string) {
	for _, line := range lines {
		if matches := aussieBBPaymentPattern.FindStringSubmatch(strings.TrimSpace(line)); len(matches) == 3 {
			return strings.TrimSpace(matches[1]), matches[2]
		}
	}
	return "", ""
}
