package tui

import (
	"fmt"
	"strings"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func EditExpense(expense *splitwise.DetailedExpense) error {
	app := tview.NewApplication()

	// Title
	title := tview.NewTextView().
		SetText(fmt.Sprintf("Editing Expense: %s (ID: %d)", expense.Description, expense.ID)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	parsedDetails := splitwise.ParseDetails(expense.Details)
	notesText := ""
	if parsedDetails != nil {
		notesText = parsedDetails.Notes
	} else {
		notesText = expense.Details
	}

	// Focus management array
	var focusables []tview.Primitive

	// Basic Info Form
	form := tview.NewForm().
		AddInputField("Description", expense.Description, 40, nil, func(text string) {
			expense.Description = text
		}).
		AddInputField("Cost", expense.Cost, 20, nil, func(text string) {
			expense.Cost = text
		}).
		AddInputField("Date", expense.Date, 25, nil, func(text string) {
			expense.Date = text
		}).
		AddTextArea("Notes", notesText, 40, 5, 0, func(text string) {
			if parsedDetails != nil {
				parsedDetails.Notes = text
			}
		})

	form.SetBorder(true).SetTitle("Basic Info").SetTitleAlign(tview.AlignLeft)
	focusables = append(focusables, form)

	// Items Table
	itemsTable := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	itemsTable.SetBorder(true).SetTitle("Items & Splits").SetTitleAlign(tview.AlignLeft)
	focusables = append(focusables, itemsTable)

	// Set Headers
	itemsTable.SetCell(0, 0, tview.NewTableCell("Description").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter).SetExpansion(1))
	itemsTable.SetCell(0, 1, tview.NewTableCell("Amount").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter).SetExpansion(1))
	itemsTable.SetCell(0, 2, tview.NewTableCell("Shared With").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter).SetExpansion(2))

	if expense.CreationMethod == "itemized" && parsedDetails != nil {
		for row, item := range parsedDetails.Items {
			itemsTable.SetCell(row+1, 0, tview.NewTableCell(item.Description).SetAlign(tview.AlignLeft))
			itemsTable.SetCell(row+1, 1, tview.NewTableCell(item.Amount).SetAlign(tview.AlignRight))
			itemsTable.SetCell(row+1, 2, tview.NewTableCell(strings.Join(item.SharedWith, ", ")).SetAlign(tview.AlignLeft))
		}
	} else {
		itemsTable.SetCell(1, 0, tview.NewTableCell("No items found").SetAlign(tview.AlignLeft))
	}

	// Split amounts table (Paid/Owed balances)
	splitTable := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	splitTable.SetBorder(true).SetTitle("Users (Paid / Owed / Net)").SetTitleAlign(tview.AlignLeft)
	focusables = append(focusables, splitTable)

	splitTable.SetCell(0, 0, tview.NewTableCell("User").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 1, tview.NewTableCell("Paid").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 2, tview.NewTableCell("Owed").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 3, tview.NewTableCell("Net").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))

	for row, eu := range expense.Users {
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))

		splitTable.SetCell(row+1, 0, tview.NewTableCell(name).SetAlign(tview.AlignLeft))
		splitTable.SetCell(row+1, 1, tview.NewTableCell(eu.PaidShare).SetAlign(tview.AlignRight))
		splitTable.SetCell(row+1, 2, tview.NewTableCell(eu.OwedShare).SetAlign(tview.AlignRight))
		splitTable.SetCell(row+1, 3, tview.NewTableCell(eu.NetBalance).SetAlign(tview.AlignRight))
	}

	// Help Text
	helpText := `Keyboard Shortcuts:
[1] P1 fully pays    | [2] P2 fully pays
[3] P1 & P2 half pay | [4] P1 pays N, P2 pays remaining N
[5] Different % / Shares / Exact amounts
[?] Toggle Help      | [Tab]/[Backtab] Switch Focus
`
	helpView := tview.NewTextView().
		SetText(helpText).
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	helpView.SetBorder(true).SetTitle("Help").SetTitleAlign(tview.AlignLeft)

	// Layout
	tablesFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(itemsTable, 0, 2, false).
		AddItem(splitTable, 0, 1, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(form, 0, 1, true).
		AddItem(tablesFlex, 0, 2, false)

	rootFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 3, 1, false).
		AddItem(mainFlex, 0, 3, true)

	// Tab Navigation logic
	currentFocusIndex := 0
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle global Tab/Backtab navigation
		if event.Key() == tcell.KeyTab {
			currentFocusIndex = (currentFocusIndex + 1) % len(focusables)
			app.SetFocus(focusables[currentFocusIndex])
			return nil
		} else if event.Key() == tcell.KeyBacktab {
			currentFocusIndex = (currentFocusIndex - 1 + len(focusables)) % len(focusables)
			app.SetFocus(focusables[currentFocusIndex])
			return nil
		}

		if event.Rune() == '?' {
			showingHelp := false
			if rootFlex.GetItemCount() == 3 {
				showingHelp = true
			}

			showingHelp = !showingHelp
			if showingHelp {
				rootFlex.AddItem(helpView, 8, 1, false)
			} else {
				rootFlex.RemoveItem(helpView)
			}
			app.Draw()
			return nil
		}

		// If itemsTable is focused, handle shortcuts
		if itemsTable.HasFocus() || splitTable.HasFocus() {
			switch event.Rune() {
			case '1':
				// TODO: Implement P1 fully pays
				return nil
			case '2':
				// TODO: Implement P2 fully pays
				return nil
			case '3':
				// TODO: Implement P1 & P2 half pay
				return nil
			case '4':
				// TODO: Implement P1 pays N, P2 pays remaining
				return nil
			case '5':
				// TODO: Implement Different % / Shares
				return nil
			}
		}

		return event
	})

	// Focus management buttons
	form.AddButton("Save (Not Implemented)", func() {
		app.Stop()
	}).
		AddButton("Quit", func() {
			app.Stop()
		})

	if err := app.SetRoot(rootFlex, true).SetFocus(form).Run(); err != nil {
		return err
	}

	return nil
}
