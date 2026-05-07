package importers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var amazonOrderIDPattern = regexp.MustCompile(`\d{3}-\d{7}-\d{7}`)

func ParseAmazonEmailText(text string) (*ParsedExpense, error) {
	lines := normalizeImportLines(text)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no Amazon email text found")
	}

	orderID := parseAmazonOrderID(lines)
	if orderID == "" {
		return nil, fmt.Errorf("could not determine Amazon order id from email text")
	}
	total := parseAmazonTotal(lines)
	if total == "" {
		return nil, fmt.Errorf("could not determine Amazon total from email text")
	}
	items, err := parseAmazonItems(lines)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("could not determine any line items from Amazon email text")
	}

	return &ParsedExpense{
		Merchant:      "Amazon Order #" + orderID,
		CurrencyCode:  "AUD",
		Total:         total,
		Notes:         buildAmazonNotes(lines, orderID),
		Items:         items,
		TaxTotal:      "0.00",
		TipTotal:      "0.00",
		SuggestedMode: ImportModeNew,
	}, nil
}

func parseAmazonOrderID(lines []string) string {
	for _, line := range lines {
		if orderID := amazonOrderIDPattern.FindString(line); orderID != "" {
			return orderID
		}
	}
	return ""
}

func parseAmazonTotal(lines []string) string {
	for _, line := range lines {
		normalized := normalizeImportMatchLine(line)
		if strings.HasPrefix(strings.ToLower(normalized), "total ") {
			if amount := extractAmazonAmount(normalized); amount != "" {
				return amount
			}
		}
	}
	return ""
}

func parseAmazonItems(lines []string) ([]ParsedLineItem, error) {
	totalIndex := -1
	for i, line := range lines {
		normalized := normalizeImportMatchLine(line)
		if strings.HasPrefix(strings.ToLower(normalized), "total ") {
			totalIndex = i
			break
		}
	}
	if totalIndex == -1 {
		return nil, fmt.Errorf("could not find Amazon total line")
	}

	var items []ParsedLineItem
	for i := 0; i < totalIndex; {
		line := strings.TrimSpace(lines[i])
		normalized := normalizeImportMatchLine(line)
		if shouldSkipAmazonLine(normalized) {
			i++
			continue
		}
		if strings.HasPrefix(strings.ToLower(normalized), "quantity:") || strings.HasPrefix(normalized, "$") {
			return nil, fmt.Errorf("unexpected Amazon item structure near %q", line)
		}

		fullTitle := line
		j := i + 1
		displayTitle := ""
		if j < totalIndex && looksLikeAmazonDisplayTitle(lines[j]) {
			displayTitle = strings.TrimSpace(lines[j])
			j++
		}
		if j >= totalIndex {
			return nil, fmt.Errorf("incomplete Amazon item block starting at %q", fullTitle)
		}

		qtyLine := normalizeImportMatchLine(lines[j])
		if !strings.HasPrefix(strings.ToLower(qtyLine), "quantity:") {
			return nil, fmt.Errorf("unknown Amazon quantity line %q for item %q", lines[j], fullTitle)
		}
		qty, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(qtyLine), "quantity:")))
		if err != nil {
			return nil, fmt.Errorf("parsing Amazon quantity from %q for item %q: %w", lines[j], fullTitle, err)
		}

		j++
		if j >= totalIndex {
			return nil, fmt.Errorf("missing Amazon price line for item %q", fullTitle)
		}
		amount := extractAmazonAmount(normalizeImportMatchLine(lines[j]))
		if amount == "" {
			return nil, fmt.Errorf("unknown Amazon price line %q for item %q", lines[j], fullTitle)
		}

		description := displayTitle
		if description == "" {
			description = simplifyAmazonTitle(fullTitle)
		}
		extra := ""
		if description != fullTitle {
			extra = fullTitle
		}
		items = append(items, ParsedLineItem{
			Description: description,
			Extra:       extra,
			Quantity:    qty,
			Amount:      amount,
		})
		i = j + 1
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Description != items[j].Description {
			return items[i].Description < items[j].Description
		}
		return items[i].Extra < items[j].Extra
	})
	return items, nil
}

func shouldSkipAmazonLine(line string) bool {
	lower := strings.ToLower(line)
	switch {
	case line == "":
		return true
	case strings.HasPrefix(lower, "arriving "):
		return true
	case strings.HasPrefix(lower, "order #"):
		return true
	case strings.EqualFold(line, "view or edit order"):
		return true
	case strings.Contains(lower, "your orders on amazon"):
		return true
	case !strings.Contains(line, "Quantity:") && !strings.Contains(line, "$") && strings.Contains(lower, ", vic"):
		return true
	}
	return false
}

func looksLikeAmazonDisplayTitle(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasSuffix(trimmed, "...") && !strings.HasPrefix(strings.ToLower(trimmed), "quantity:")
}

func simplifyAmazonTitle(fullTitle string) string {
	trimmed := strings.TrimSpace(fullTitle)
	if utf8.RuneCountInString(trimmed) <= 40 {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:37]) + "..."
}

func extractAmazonAmount(line string) string {
	idx := strings.Index(line, "$")
	if idx == -1 {
		return ""
	}
	raw := strings.TrimSpace(line[idx+1:])
	raw = strings.ReplaceAll(raw, ",", "")
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ".") {
		if _, err := strconv.ParseFloat(raw, 64); err != nil {
			return ""
		}
		return raw
	}
	if _, err := strconv.Atoi(raw); err != nil {
		return ""
	}
	if len(raw) == 1 {
		return "0.0" + raw
	}
	if len(raw) == 2 {
		return "0." + raw
	}
	return raw[:len(raw)-2] + "." + raw[len(raw)-2:]
}

func buildAmazonNotes(lines []string, orderID string) string {
	notes := []string{"Imported from Amazon email text", "Order # " + orderID}
	for i, line := range lines {
		normalized := normalizeImportMatchLine(line)
		if strings.HasPrefix(strings.ToLower(normalized), "arriving ") {
			notes = append(notes, normalized)
			if i+1 < len(lines) {
				next := normalizeImportMatchLine(lines[i+1])
				if next != "" && !strings.Contains(strings.ToLower(next), "order #") {
					notes = append(notes, next)
				}
			}
			break
		}
	}
	return strings.Join(notes, "\n")
}
