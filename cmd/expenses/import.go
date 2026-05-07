package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/importers"
	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
	"github.com/arran4/personal-splitwise-helper-scripts/pkg/tui"
)

func handleImport(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: expenses import doordash email text [--mode auto|new|update] [--stdin] [--id <expense-id>] [--group-id <id>|--friend-id <id>]")
	}

	provider := strings.ToLower(strings.TrimSpace(args[0]))
	channel := strings.ToLower(strings.TrimSpace(args[1]))
	format := strings.ToLower(strings.TrimSpace(args[2]))
	if provider != "doordash" || channel != "email" || format != "text" {
		return fmt.Errorf("unsupported import source %q %q %q", provider, channel, format)
	}

	importCmd := flag.NewFlagSet("import", flag.ContinueOnError)
	importCmd.SetOutput(io.Discard)

	mode := importCmd.String("mode", "auto", "Import mode: auto, new, or update")
	fromStdin := importCmd.Bool("stdin", false, "Read import text from stdin")
	id := importCmd.String("id", "", "Existing expense ID to update")
	groupID := importCmd.Int("group-id", 0, "Create the imported expense in this group")
	friendID := importCmd.Int("friend-id", 0, "Create the imported expense with this friend")
	verbose := importCmd.Bool("verbose", false, "Print the full server success payload after send")
	limit := importCmd.Int("limit", 20, "Number of recent expenses to fetch per page when resolving updates")
	offset := importCmd.Int("offset", 0, "Initial offset when resolving updates")
	pages := importCmd.String("pages", "", "Page selection when resolving updates: N, N-M, N-, or all")

	if err := importCmd.Parse(args[3:]); err != nil {
		return err
	}
	if *groupID != 0 && *friendID != 0 {
		return fmt.Errorf("provide only one of --group-id or --friend-id")
	}

	text, err := readImportText(*fromStdin)
	if err != nil {
		return err
	}
	parsed, err := importers.ParseDoorDashEmailText(text)
	if err != nil {
		return err
	}

	updateOpts := expenseListOptions{
		limit:  *limit,
		offset: *offset,
		pages:  *pages,
	}

	explicitMode := strings.ToLower(strings.TrimSpace(*mode))
	switch explicitMode {
	case "", "auto", "new", "update":
	default:
		return fmt.Errorf("invalid --mode value %q", *mode)
	}

	updateID := strings.TrimSpace(*id)
	resolvedMode := explicitMode
	if resolvedMode == "" || resolvedMode == "auto" {
		if updateID != "" {
			resolvedMode = "update"
		} else if parsed.SuggestedMode == importers.ImportModeUpdate {
			if autoID, ok, err := autoSelectImportedExpense(parsed, updateOpts); err != nil {
				return err
			} else if ok {
				updateID = autoID
				resolvedMode = "update"
			} else {
				resolvedMode = "new"
			}
		} else {
			resolvedMode = "new"
		}
	}

	var expense *splitwise.DetailedExpense
	updatedExisting := false
	if resolvedMode == "update" {
		if updateID == "" {
			selectedID, err := chooseRecentExpense(updateOpts)
			if err != nil {
				return fmt.Errorf("selecting expense to update: %w", err)
			}
			updateID = selectedID
		}
		resp, err := fetchExpense(updateID, false)
		if err != nil {
			return fmt.Errorf("fetching expense %s: %w", updateID, err)
		}
		expense = &resp.Expense
		updatedExisting = true
	} else {
		selectedGroupID := *groupID
		selectedFriendID := *friendID
		if selectedGroupID == 0 && selectedFriendID == 0 {
			var err error
			selectedGroupID, selectedFriendID, err = chooseExpenseContext()
			if err != nil {
				return fmt.Errorf("choosing expense context: %w", err)
			}
		}
		expense, err = initializeNewExpense(selectedGroupID, selectedFriendID)
		if err != nil {
			return fmt.Errorf("creating imported expense draft: %w", err)
		}
	}

	if err := applyImportedExpense(expense, parsed, updatedExisting); err != nil {
		return err
	}

	sent, sendResponse, err := tui.EditExpense(expense)
	if err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}
	if !sent {
		return nil
	}
	if updatedExisting && updateID != "" {
		if err := invalidateExpenseCache(updateID); err != nil {
			fmt.Println("Warning: could not invalidate cache:", err)
		}
	}
	if *verbose && len(sendResponse) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, sendResponse, "", "  "); err == nil {
			fmt.Println(pretty.String())
		} else {
			fmt.Println(string(sendResponse))
		}
	} else {
		fmt.Println("success")
	}
	return nil
}

func readImportText(forceStdin bool) (string, error) {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("inspecting stdin: %w", err)
	}
	stdinPiped := (stdinInfo.Mode() & os.ModeCharDevice) == 0
	if !forceStdin && !stdinPiped {
		return "", fmt.Errorf("no import input provided; use --stdin and pipe DoorDash email text into the command")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("stdin import text is empty")
	}
	return text, nil
}

func autoSelectImportedExpense(parsed *importers.ParsedExpense, opts expenseListOptions) (string, bool, error) {
	if parsed == nil || strings.TrimSpace(parsed.Merchant) == "" {
		return "", false, nil
	}
	expenses, err := fetchExpensesPageSet(opts)
	if err != nil {
		return "", false, err
	}

	target := normalizeImportMatchText(parsed.Merchant)
	var matches []splitwise.Expense
	for _, expense := range expenses {
		if normalizeImportMatchText(expense.Description) == target {
			matches = append(matches, expense)
		}
	}
	if len(matches) == 1 {
		return strconv.Itoa(matches[0].ID), true, nil
	}
	return "", false, nil
}

func normalizeImportMatchText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("-", " ", "_", " ", ",", " ", ".", " ", "  ", " ")
	s = replacer.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func applyImportedExpense(expense *splitwise.DetailedExpense, parsed *importers.ParsedExpense, preserveExistingNotes bool) error {
	if expense == nil {
		return fmt.Errorf("expense is nil")
	}
	if parsed == nil {
		return fmt.Errorf("parsed import is nil")
	}

	previousDetails := splitwise.ParseDetails(expense.Details)
	previousTotal := itemizedDetailTotal(previousDetails)
	newDetails := buildImportedDetails(expense, parsed, previousDetails, preserveExistingNotes)
	newTotal := itemizedDetailTotal(newDetails)

	expense.Description = parsed.Merchant
	expense.Details = splitwise.SerializeDetails(newDetails)
	if parsed.CurrencyCode != "" {
		expense.CurrencyCode = parsed.CurrencyCode
	}
	if expense.CurrencyCode == "" {
		expense.CurrencyCode = "AUD"
	}
	if parsed.Total != "" {
		expense.Cost = parsed.Total
	} else {
		expense.Cost = formatImportMoney(newTotal)
	}
	if expense.Date == "" {
		expense.Date = time.Now().Format("2006-01-02")
	}

	splitwise.AutoCorrectPaidShares(expense, previousTotal, newTotal)
	splitwise.CalculateOwed(expense, newDetails)
	return nil
}

func buildImportedDetails(expense *splitwise.DetailedExpense, parsed *importers.ParsedExpense, previousDetails *splitwise.ItemizedDetail, preserveExistingNotes bool) *splitwise.ItemizedDetail {
	names := expenseParticipantNames(expense)
	notes := parsed.Notes
	if preserveExistingNotes && previousDetails != nil && strings.TrimSpace(previousDetails.Notes) != "" {
		notes = previousDetails.Notes
	}

	details := &splitwise.ItemizedDetail{
		Notes: notes,
	}
	for _, item := range parsed.Items {
		details.Items = append(details.Items, splitwise.Item{
			Description: splitwise.FormatItemDescription(item.Quantity, item.Description),
			Amount:      item.Amount,
			SharedWith:  append([]string(nil), names...),
		})
	}
	for _, fee := range parsed.Fees {
		details.Items = append(details.Items, splitwise.Item{
			Description: splitwise.FormatItemDescription(fee.Quantity, fee.Description),
			Amount:      fee.Amount,
			SharedWith:  append([]string(nil), names...),
		})
	}
	details.Tax = splitImportAmountAcrossUsers(parsed.TaxTotal, names)
	details.Tip = splitImportAmountAcrossUsers(parsed.TipTotal, names)
	return details
}

func expenseParticipantNames(expense *splitwise.DetailedExpense) []string {
	names := make([]string, 0, len(expense.Users))
	for _, eu := range expense.Users {
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func splitImportAmountAcrossUsers(amount string, names []string) []splitwise.PersonAmount {
	if len(names) == 0 {
		return nil
	}
	cents := parseImportAmountCents(amount)
	if cents <= 0 {
		out := make([]splitwise.PersonAmount, 0, len(names))
		for _, name := range names {
			out = append(out, splitwise.PersonAmount{Name: name, Amount: "0.00"})
		}
		return out
	}

	base := cents / len(names)
	remainder := cents % len(names)
	out := make([]splitwise.PersonAmount, 0, len(names))
	for i, name := range names {
		share := base
		if i == len(names)-1 {
			share += remainder
		}
		out = append(out, splitwise.PersonAmount{
			Name:   name,
			Amount: formatImportMoney(float64(share) / 100.0),
		})
	}
	return out
}

func parseImportAmountCents(amount string) int {
	parts := strings.SplitN(strings.TrimSpace(amount), ".", 3)
	if len(parts) == 0 || parts[0] == "" {
		return 0
	}
	whole, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	cents := 0
	if len(parts) > 1 {
		frac := parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) > 2 {
			frac = frac[:2]
		}
		cents, _ = strconv.Atoi(frac)
	}
	return (whole * 100) + cents
}

func itemizedDetailTotal(details *splitwise.ItemizedDetail) float64 {
	if details == nil {
		return 0
	}
	total := 0.0
	for _, item := range details.Items {
		total += floatFromMoney(item.Amount)
	}
	for _, item := range details.Tax {
		total += floatFromMoney(item.Amount)
	}
	for _, item := range details.Tip {
		total += floatFromMoney(item.Amount)
	}
	return total
}

func floatFromMoney(value string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return f
}

func formatImportMoney(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}
