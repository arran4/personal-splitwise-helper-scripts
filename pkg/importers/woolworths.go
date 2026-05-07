package importers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var woolworthsItemPattern = regexp.MustCompile(`^(.+?)\s+([0-9]+(?:\.[0-9]{2})?)\s+([0-9]+(?:\.[0-9]{2})?)\s+([0-9]+(?:\.[0-9]{2})?)$`)

func ParseWoolworthsEmailText(text string) (*ParsedExpense, error) {
	lines := normalizeImportLines(text)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no Woolworths email text found")
	}
	if !looksLikeWoolworthsEmail(lines) {
		return nil, fmt.Errorf("text does not look like a Woolworths order email")
	}
	orderNumber := parseWoolworthsOrderNumber(lines)
	if orderNumber == "" {
		return nil, fmt.Errorf("could not determine Woolworths order number from email text")
	}

	total := parseWoolworthsSummaryAmount(lines, "estimated amount to be charged:")
	if total == "" {
		return nil, fmt.Errorf("could not determine Woolworths total from email text")
	}
	items, err := parseWoolworthsItems(lines)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("could not determine any line items from Woolworths email text")
	}

	return &ParsedExpense{
		Merchant:      "Woolworths Order #" + orderNumber,
		CurrencyCode:  "AUD",
		Total:         total,
		Notes:         buildWoolworthsNotes(lines),
		Items:         items,
		Fees:          parseWoolworthsFees(lines),
		TaxTotal:      "0.00",
		TipTotal:      "0.00",
		SuggestedMode: ImportModeNew,
	}, nil
}

func looksLikeWoolworthsEmail(lines []string) bool {
	hasOrderNumber := false
	hasItems := false
	hasTotal := false
	for _, line := range lines {
		lower := strings.ToLower(normalizeImportMatchLine(line))
		switch {
		case strings.HasPrefix(lower, "order number:"):
			hasOrderNumber = true
		case lower == "your items":
			hasItems = true
		case strings.HasPrefix(lower, "estimated amount to be charged:"):
			hasTotal = true
		}
	}
	return hasOrderNumber && hasItems && hasTotal
}

func parseWoolworthsOrderNumber(lines []string) string {
	for _, line := range lines {
		normalized := normalizeImportMatchLine(line)
		lower := strings.ToLower(normalized)
		if strings.HasPrefix(lower, "order number:") {
			return strings.TrimSpace(normalized[len("Order number:"):])
		}
	}
	return ""
}

func parseWoolworthsSummaryAmount(lines []string, label string) string {
	for _, line := range lines {
		normalized := normalizeImportMatchLine(line)
		lower := strings.ToLower(normalized)
		if strings.HasPrefix(lower, label) {
			fields := strings.Fields(normalized)
			if len(fields) == 0 {
				continue
			}
			last := fields[len(fields)-1]
			if _, err := strconv.ParseFloat(last, 64); err == nil {
				return last
			}
		}
	}
	return ""
}

func parseWoolworthsItems(lines []string) ([]ParsedLineItem, error) {
	start := -1
	for i, line := range lines {
		if strings.EqualFold(normalizeImportMatchLine(line), "Woolworths items") {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("could not find Woolworths items section")
	}

	var items []ParsedLineItem
	for i := start; i < len(lines); i++ {
		line := normalizeImportMatchLine(lines[i])
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "subtotal:") || strings.HasPrefix(lower, "delivery fee:") || strings.HasPrefix(lower, "paper bags:") || strings.HasPrefix(lower, "estimated amount to be charged:") || strings.HasPrefix(lower, "paid with ") {
			break
		}

		matches := woolworthsItemPattern.FindStringSubmatch(line)
		if len(matches) != 5 {
			return nil, fmt.Errorf("unknown Woolworths item line %q", lines[i])
		}
		qtyFloat, err := strconv.ParseFloat(matches[3], 64)
		if err != nil {
			return nil, fmt.Errorf("parsing Woolworths quantity from %q: %w", lines[i], err)
		}
		qty := 1
		extraLines := []string{}
		if qtyFloat == float64(int(qtyFloat)) {
			qty = int(qtyFloat)
		} else {
			extraLines = append(extraLines, "Unit price: "+matches[2], "Qty: "+matches[3])
		}
		items = append(items, ParsedLineItem{
			Description: strings.TrimSpace(matches[1]),
			Extra:       strings.Join(extraLines, "\n"),
			Quantity:    qty,
			Amount:      matches[4],
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Description < items[j].Description
	})
	return items, nil
}

func parseWoolworthsFees(lines []string) []ParsedLineItem {
	type feeDef struct {
		label string
		name  string
	}
	defs := []feeDef{
		{label: "delivery fee:", name: "Delivery Fee"},
		{label: "paper bags:", name: "Paper Bags"},
	}
	var fees []ParsedLineItem
	for _, def := range defs {
		amount := parseWoolworthsSummaryAmount(lines, def.label)
		if amount == "" || amount == "0.00" {
			continue
		}
		fees = append(fees, ParsedLineItem{Description: def.name, Quantity: 1, Amount: amount})
	}
	return fees
}

func buildWoolworthsNotes(lines []string) string {
	notes := []string{"Imported from Woolworths email text"}
	for _, field := range []struct {
		prefix string
		label  string
	}{
		{prefix: "order number:", label: "Order Number"},
	} {
		for _, line := range lines {
			lower := strings.ToLower(normalizeImportMatchLine(line))
			if strings.HasPrefix(lower, field.prefix) {
				value := strings.TrimSpace(normalizeImportMatchLine(line)[len(field.prefix):])
				notes = append(notes, fmt.Sprintf("%s: %s", field.label, value))
				break
			}
		}
	}

	for i, line := range lines {
		if strings.HasPrefix(strings.ToLower(normalizeImportMatchLine(line)), "order number:") {
			if i+1 < len(lines) {
				address := normalizeImportMatchLine(lines[i+1])
				if address != "" {
					notes = append(notes, "Delivery: "+address)
				}
			}
			if i+2 < len(lines) {
				dateLine := normalizeImportMatchLine(lines[i+2])
				if dateLine != "" {
					notes = append(notes, dateLine)
				}
			}
			if i+3 < len(lines) {
				timeLine := normalizeImportMatchLine(lines[i+3])
				if timeLine != "" {
					notes = append(notes, timeLine)
				}
			}
			break
		}
	}
	return strings.Join(notes, "\n")
}
