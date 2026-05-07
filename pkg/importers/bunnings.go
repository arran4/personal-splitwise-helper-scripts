package importers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	bunningsItemLinePattern = regexp.MustCompile(`^(?:(.+?)\s+)?(\d+)\s+\$([0-9]+(?:\.[0-9]{2})?)$`)
	bunningsInPattern       = regexp.MustCompile(`^I/N:\s*([0-9]+)$`)
)

func normalizeImportMatchLine(line string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
}

func ParseBunningsEmailText(text string) (*ParsedExpense, error) {
	lines := normalizeImportLines(text)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no Bunnings email text found")
	}
	if !looksLikeBunningsEmail(lines) {
		return nil, fmt.Errorf("text does not look like a Bunnings order email")
	}

	total := parseBunningsSummaryAmount(lines, "order total")
	if total == "" {
		return nil, fmt.Errorf("could not determine order total from Bunnings email text")
	}

	items, locations, err := parseBunningsItems(lines)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("could not determine any line items from Bunnings email text")
	}

	parsed := &ParsedExpense{
		Merchant:      "Bunnings",
		CurrencyCode:  "AUD",
		Total:         total,
		Items:         items,
		TaxTotal:      "0.00",
		TipTotal:      "0.00",
		SuggestedMode: ImportModeNew,
		Notes:         buildBunningsNotes(lines, locations),
	}
	return parsed, nil
}

func looksLikeBunningsEmail(lines []string) bool {
	hasOrderDetails := false
	hasOrderNumber := false
	hasFulfillment := false
	hasItemHeader := false
	hasOrderTotal := false
	for _, line := range lines {
		lower := strings.ToLower(normalizeImportMatchLine(line))
		switch {
		case strings.HasPrefix(lower, "your order details"):
			hasOrderDetails = true
		case strings.HasPrefix(lower, "order number:"):
			hasOrderNumber = true
		case strings.HasPrefix(lower, "delivery to:") || strings.HasPrefix(lower, "collection from:"):
			hasFulfillment = true
		case lower == "item due qty total":
			hasItemHeader = true
		case strings.HasPrefix(lower, "order total"):
			hasOrderTotal = true
		}
	}
	if hasFulfillment && hasItemHeader && hasOrderTotal && hasOrderNumber {
		return true
	}
	return hasOrderDetails && hasOrderNumber && hasFulfillment
}

func parseBunningsSummaryAmount(lines []string, label string) string {
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if lower == label || strings.HasPrefix(lower, label+" ") {
			if amount := extractDoorDashAmount(line); amount != "" {
				return amount
			}
		}
	}
	return ""
}

func parseBunningsItems(lines []string) ([]ParsedLineItem, []string, error) {
	type section struct {
		location string
		start    int
	}

	var sections []section
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "delivery to:") || strings.HasPrefix(lower, "collection from:") {
			sections = append(sections, section{location: line, start: i})
		}
	}
	if len(sections) == 0 {
		return nil, nil, fmt.Errorf("could not determine any fulfilment sections from Bunnings email text")
	}

	var items []ParsedLineItem
	var locations []string
	for idx, sec := range sections {
		locations = append(locations, sec.location)
		end := len(lines)
		if idx+1 < len(sections) {
			end = sections[idx+1].start
		}
		sectionItems, err := parseBunningsFulfilmentSection(lines[sec.start:end], sec.location)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, sectionItems...)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Description != items[j].Description {
			return items[i].Description < items[j].Description
		}
		return items[i].Extra < items[j].Extra
	})
	return items, locations, nil
}

func parseBunningsFulfilmentSection(lines []string, location string) ([]ParsedLineItem, error) {
	headerIndex := -1
	for i, line := range lines {
		if strings.EqualFold(normalizeImportMatchLine(line), "Item Due Qty Total") {
			headerIndex = i
			break
		}
	}
	if headerIndex == -1 {
		return nil, fmt.Errorf("could not find item table header for %q", location)
	}

	var items []ParsedLineItem
	for i := headerIndex + 1; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "order total") || strings.HasPrefix(lower, "amount paid") || strings.HasPrefix(lower, "amount outstanding") || strings.HasPrefix(lower, "payment method") || strings.HasPrefix(lower, "amount other") {
			break
		}
		if strings.HasPrefix(lower, "delivery to:") || strings.HasPrefix(lower, "collection from:") {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "w") && strings.Contains(line, "-") {
			i++
			continue
		}
		if strings.EqualFold(normalizeImportMatchLine(line), "Item Due Qty Total") {
			i++
			continue
		}

		if i+2 >= len(lines) {
			return nil, fmt.Errorf("incomplete Bunnings item block starting at %q", line)
		}

		description := line
		inLine := strings.TrimSpace(lines[i+1])
		if !bunningsInPattern.MatchString(inLine) {
			return nil, fmt.Errorf("unknown Bunnings item detail line %q for item %q", inLine, description)
		}
		detailLine := strings.TrimSpace(lines[i+2])
		matches := bunningsItemLinePattern.FindStringSubmatch(detailLine)
		if len(matches) != 4 {
			return nil, fmt.Errorf("unknown Bunnings quantity/amount line %q for item %q", detailLine, description)
		}

		qty, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("parsing Bunnings quantity %q for item %q: %w", matches[2], description, err)
		}
		extraLines := []string{inLine, location}
		due := strings.TrimSpace(matches[1])
		if due != "" {
			extraLines = append(extraLines, "Due: "+due)
		}
		items = append(items, ParsedLineItem{
			Description: description,
			Extra:       strings.Join(extraLines, "\n"),
			Quantity:    qty,
			Amount:      matches[3],
		})
		i += 3
	}
	return items, nil
}

func buildBunningsNotes(lines, locations []string) string {
	notes := []string{"Imported from Bunnings email text"}
	for _, field := range []struct {
		prefix string
		label  string
	}{
		{prefix: "order date:", label: "Order Date"},
		{prefix: "order number:", label: "Order Number"},
	} {
		for _, line := range lines {
			if strings.HasPrefix(strings.ToLower(line), field.prefix) {
				value := strings.TrimSpace(line[len(field.prefix):])
				notes = append(notes, fmt.Sprintf("%s: %s", field.label, value))
				break
			}
		}
	}
	for _, location := range locations {
		notes = append(notes, location)
	}
	return strings.Join(notes, "\n")
}
