package importers

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type ImportMode string

const (
	ImportModeNew    ImportMode = "new"
	ImportModeUpdate ImportMode = "update"
)

type ParsedLineItem struct {
	Description string
	Quantity    int
	Amount      string
}

type ParsedExpense struct {
	Merchant      string
	CurrencyCode  string
	Total         string
	Notes         string
	Items         []ParsedLineItem
	Fees          []ParsedLineItem
	TaxTotal      string
	TipTotal      string
	SuggestedMode ImportMode
}

var (
	doordashQtyPattern    = regexp.MustCompile(`^(\d+)x\s+(.+)$`)
	doordashAmountPattern = regexp.MustCompile(`(?:A\$|\$)\s*([0-9]+(?:\.[0-9]{2})?)`)
)

func ParseDoorDashEmailText(text string) (*ParsedExpense, error) {
	lines := normalizeImportLines(text)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no DoorDash email text found")
	}

	parsed := &ParsedExpense{
		CurrencyCode:  detectCurrencyCode(lines),
		SuggestedMode: detectDoorDashMode(lines),
	}
	parsed.Merchant = parseDoorDashMerchant(lines)
	parsed.Total = parseDoorDashTotal(lines)
	parsed.TaxTotal = parseDoorDashSummaryAmount(lines, []string{"tax", "taxes"})
	parsed.TipTotal = parseDoorDashSummaryAmount(lines, []string{"tip", "dasher tip"})
	parsed.Fees = parseDoorDashFees(lines)
	parsed.Items = parseDoorDashItems(lines)
	parsed.Notes = buildDoorDashNotes(lines)

	if parsed.Merchant == "" {
		return nil, fmt.Errorf("could not determine merchant from DoorDash email text")
	}
	if parsed.Total == "" {
		return nil, fmt.Errorf("could not determine total from DoorDash email text")
	}
	if len(parsed.Items) == 0 {
		return nil, fmt.Errorf("could not determine any line items from DoorDash email text")
	}
	if parsed.CurrencyCode == "" {
		parsed.CurrencyCode = "AUD"
	}
	return parsed, nil
}

func normalizeImportLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, raw := range rawLines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func detectCurrencyCode(lines []string) string {
	for _, line := range lines {
		if strings.Contains(line, "A$") {
			return "AUD"
		}
	}
	return ""
}

func detectDoorDashMode(lines []string) ImportMode {
	for _, line := range lines {
		lower := strings.ToLower(line)
		if lower == "items that were adjusted" || lower == "items you ordered" || lower == "final total charged" {
			return ImportModeUpdate
		}
	}
	return ImportModeNew
}

func parseDoorDashMerchant(lines []string) string {
	for i, line := range lines {
		if !strings.HasPrefix(strings.ToLower(line), "paid with ") {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			candidate := strings.TrimSpace(lines[j])
			if candidate == "" || strings.EqualFold(candidate, "your receipt") {
				continue
			}
			if strings.HasPrefix(strings.ToLower(candidate), "total:") {
				continue
			}
			if isDoorDashSectionLabel(candidate) {
				continue
			}
			if extractDoorDashAmount(candidate) != "" && candidate == parseDoorDashTotal(lines) {
				continue
			}
			return candidate
		}
	}

	for _, line := range lines {
		if strings.EqualFold(line, "your receipt") || strings.HasPrefix(strings.ToLower(line), "total:") || strings.HasPrefix(strings.ToLower(line), "paid with ") {
			continue
		}
		if isDoorDashSectionLabel(line) {
			continue
		}
		return line
	}
	return ""
}

func parseDoorDashTotal(lines []string) string {
	for _, labels := range [][]string{
		{"final total charged"},
		{"total charged"},
		{"total:"},
	} {
		if amount := parseDoorDashSummaryAmount(lines, labels); amount != "" {
			return amount
		}
	}
	return ""
}

func parseDoorDashSummaryAmount(lines []string, labels []string) string {
	for i, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		for _, label := range labels {
			if lower == label || strings.HasPrefix(lower, label+" ") {
				if amount := extractDoorDashAmount(line); amount != "" {
					return amount
				}
				for j := i + 1; j < len(lines); j++ {
					if amount := extractDoorDashAmount(lines[j]); amount != "" {
						return amount
					}
					if isDoorDashSectionLabel(lines[j]) || doordashQtyPattern.MatchString(lines[j]) {
						break
					}
				}
			}
		}
	}
	return ""
}

func parseDoorDashFees(lines []string) []ParsedLineItem {
	type feeLabel struct {
		match string
		name  string
	}
	labels := []feeLabel{
		{match: "bag fee", name: "Bag Fee"},
		{match: "delivery fee", name: "Delivery Fee"},
		{match: "service fee", name: "Service Fee"},
		{match: "regulatory response fee", name: "Regulatory Response Fee"},
	}

	var fees []ParsedLineItem
	for _, label := range labels {
		amount := parseDoorDashSummaryAmount(lines, []string{label.match})
		if amount == "" {
			continue
		}
		if amount == "0.00" {
			continue
		}
		fees = append(fees, ParsedLineItem{
			Description: label.name,
			Quantity:    1,
			Amount:      amount,
		})
	}
	return fees
}

func extractItemsFromSection(lines []string, startIndex int) []ParsedLineItem {
	var items []ParsedLineItem
	for i := startIndex; i < len(lines); i++ {
		line := lines[i]
		if isDoorDashSummaryLabel(line) || isDoorDashSectionLabel(line) {
			break
		}
		matches := doordashQtyPattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		qty, _ := strconv.Atoi(matches[1])
		description := strings.TrimSpace(matches[2])
		if qty <= 0 || description == "" {
			continue
		}

		amount := ""
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			if doordashQtyPattern.MatchString(next) || isDoorDashSummaryLabel(next) || isDoorDashSectionLabel(next) {
				break
			}
			if parsed := extractDoorDashAmount(next); parsed != "" {
				amount = parsed
				i = j // Move parser forward
				break
			}
		}
		if amount == "" {
			continue
		}

		items = append(items, ParsedLineItem{
			Description: description,
			Quantity:    qty,
			Amount:      amount,
		})
	}
	return items
}

func parseDoorDashItems(lines []string) []ParsedLineItem {
	itemMap := make(map[string]ParsedLineItem)

	// 1. Parse baseline items from "Items you ordered" or "Your receipt"
	orderedItemsStart := -1
	for i, line := range lines {
		if strings.EqualFold(line, "Items you ordered") {
			orderedItemsStart = i + 1
			break
		}
	}
	if orderedItemsStart == -1 {
		for i, line := range lines {
			if strings.EqualFold(line, "Your receipt") {
				orderedItemsStart = i + 1
				break
			}
		}
	}

	if orderedItemsStart != -1 {
		orderedItems := extractItemsFromSection(lines, orderedItemsStart)
		for _, item := range orderedItems {
			itemMap[item.Description] = item
		}
	}

	// 2. Process "Items that were adjusted" for substitutions
	adjustedItemsStart := -1
	for i, line := range lines {
		if strings.EqualFold(line, "Items that were adjusted") {
			adjustedItemsStart = i + 1
			break
		}
	}

	if adjustedItemsStart != -1 {
		for i := adjustedItemsStart; i < len(lines); i++ {
			line := lines[i]
			if isDoorDashSummaryLabel(line) {
				break
			}
			if strings.EqualFold(line, "Substituted with:") {
				// The substituted item is on the next lines
				subItems := extractItemsFromSection(lines, i+1)
				if len(subItems) > 0 {
					subItem := subItems[0]
					if existing, ok := itemMap[subItem.Description]; ok {
						existing.Quantity += subItem.Quantity
						existingAmount, _ := strconv.ParseFloat(existing.Amount, 64)
						subAmount, _ := strconv.ParseFloat(subItem.Amount, 64)
						existing.Amount = fmt.Sprintf("%.2f", existingAmount+subAmount)
						itemMap[subItem.Description] = existing
					} else {
						itemMap[subItem.Description] = subItem
					}
				}
			}
		}
	}

	var finalItems []ParsedLineItem
	for _, item := range itemMap {
		finalItems = append(finalItems, item)
	}
	sort.Slice(finalItems, func(i, j int) bool {
		return finalItems[i].Description < finalItems[j].Description
	})

	return finalItems
}

func buildDoorDashNotes(lines []string) string {
	notes := []string{"Imported from DoorDash email text"}

	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "paid with ") {
			notes = append(notes, line)
			break
		}
	}

	receiptIndex := -1
	for i, line := range lines {
		if strings.EqualFold(line, "Your receipt") {
			receiptIndex = i
			break
		}
	}
	if receiptIndex >= 0 {
		for i := receiptIndex + 1; i < len(lines); i++ {
			line := lines[i]
			if strings.HasPrefix(line, "- For:") || isDoorDashSectionLabel(line) || doordashQtyPattern.MatchString(line) {
				break
			}
			if extractDoorDashAmount(line) != "" {
				continue
			}
			notes = append(notes, line)
			break
		}
	}

	return strings.Join(notes, "\n")
}

func extractDoorDashAmount(line string) string {
	matches := doordashAmountPattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

func isDoorDashSectionLabel(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch lower {
	case "your receipt", "items that were adjusted", "out of stock", "items you ordered", "get order help", "substituted", "substituted with:":
		return true
	}
	return isDoorDashSummaryLabel(line)
}

func isDoorDashSummaryLabel(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch lower {
	case "subtotal", "tax", "taxes", "bag fee", "delivery fee", "delivery fee ", "service fee", "dasher tip", "tip", "final total charged", "total charged", "regulatory response fee":
		return true
	}
	return strings.HasPrefix(lower, "total:")
}
