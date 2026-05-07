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

	// Focus management buttons
	form.AddButton("Save (Not Implemented)", func() {
		app.Stop()
	}).
		AddButton("Quit", func() {
			app.Stop()
		})

	// Items Table
	itemsTable := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	itemsTable.SetBorder(true).SetTitle("Items & Splits (Press Enter on an item)").SetTitleAlign(tview.AlignLeft)

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

	// Split amounts table (Paid/Owed balances) - Total Summary
	splitTable := tview.NewTable().
		SetBorders(true).
		SetSelectable(false, false).
		SetFixed(1, 0)

	splitTable.SetBorder(true).SetTitle("Users (Paid / Owed / Net)").SetTitleAlign(tview.AlignLeft)

	splitTable.SetCell(0, 0, tview.NewTableCell("User").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 1, tview.NewTableCell("Paid").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 2, tview.NewTableCell("Owed").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 3, tview.NewTableCell("Net").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))

	var p1, p2 string = "P1", "P2"
	for row, eu := range expense.Users {
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))

		if row == 0 {
			p1 = name
		} else if row == 1 {
			p2 = name
		}

		splitTable.SetCell(row+1, 0, tview.NewTableCell(name).SetAlign(tview.AlignLeft))
		splitTable.SetCell(row+1, 1, tview.NewTableCell(eu.PaidShare).SetAlign(tview.AlignRight))
		splitTable.SetCell(row+1, 2, tview.NewTableCell(eu.OwedShare).SetAlign(tview.AlignRight))
		splitTable.SetCell(row+1, 3, tview.NewTableCell(eu.NetBalance).SetAlign(tview.AlignRight))
	}

	// Focus management array - gather all focusable elements directly
	var focusables []tview.Primitive
	for i := 0; i < form.GetFormItemCount(); i++ {
		focusables = append(focusables, form.GetFormItem(i))
	}
	for i := 0; i < form.GetButtonCount(); i++ {
		focusables = append(focusables, form.GetButton(i))
	}
	focusables = append(focusables, itemsTable)

	// Help Text
	helpText := `Keyboard Shortcuts:
[?] Toggle Help      | [Tab]/[Backtab] Switch Focus
(Press Enter on an item to see split options)
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

	pages := tview.NewPages()
	pages.AddPage("main", rootFlex, true, true)

	isModalOpen := false

	// Modal action handling
	itemsTable.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return
		}
		if expense.CreationMethod != "itemized" || parsedDetails == nil || row-1 >= len(parsedDetails.Items) {
			return
		}

		item := parsedDetails.Items[row-1]

		list := tview.NewList().ShowSecondaryText(true)
		list.SetBorder(true).SetTitle(fmt.Sprintf("Actions: %s", item.Description))

		closeModal := func() {
			pages.RemovePage("split_modal")
			isModalOpen = false
			app.SetFocus(itemsTable)
		}

		showPromptForm := func(title string, fields []string, initialValues []string, onSubmit func(values []string)) {
			promptForm := tview.NewForm()
			for i, f := range fields {
				initVal := ""
				if i < len(initialValues) {
					initVal = initialValues[i]
				}
				promptForm.AddInputField(f, initVal, 20, nil, nil)
			}
			promptForm.AddButton("Save", func() {
				var vals []string
				for i := 0; i < len(fields); i++ {
					vals = append(vals, promptForm.GetFormItem(i).(*tview.InputField).GetText())
				}
				onSubmit(vals)
				pages.RemovePage("prompt_modal")
				closeModal()
			})
			promptForm.AddButton("Cancel", func() {
				pages.RemovePage("prompt_modal")
				app.SetFocus(list)
			})
			promptForm.SetBorder(true).SetTitle(title)

			modalForm := tview.NewFlex().
				AddItem(nil, 0, 1, false).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(nil, 0, 1, false).
					AddItem(promptForm, 15, 1, true).
					AddItem(nil, 0, 1, false), 60, 1, true).
				AddItem(nil, 0, 1, false)

			pages.AddPage("prompt_modal", modalForm, true, true)
			app.SetFocus(promptForm)
		}

		list.AddItem(fmt.Sprintf("%s fully pays", p1), "", '1', func() {
			// TODO: Implement P1 fully pays
			closeModal()
		})
		list.AddItem(fmt.Sprintf("%s fully pays", p2), "", '2', func() {
			// TODO: Implement P2 fully pays
			closeModal()
		})
		list.AddItem("Both half pay for all (50/50)", "", '3', func() {
			// TODO: Implement Half Pay
			closeModal()
		})
		list.AddItem(fmt.Sprintf("%s pays N items, %s pays remaining", p1, p2), "Qty split based on '1x ' prefix", '4', func() {
			showPromptForm("Split N Items", []string{"Items for " + p1, "Items for " + p2}, nil, func(vals []string) {
				// TODO: Implement Qty Split
			})
		})
		list.AddItem("Different %", "Prompt for percentage", '5', func() {
			showPromptForm("Percentage Split", []string{"% for " + p1, "% for " + p2}, nil, func(vals []string) {
				// TODO: Implement Percent Split
			})
		})
		list.AddItem("Shares", "Prompt for shares", '6', func() {
			showPromptForm("Shares Split", []string{"Shares for " + p1, "Shares for " + p2}, nil, func(vals []string) {
				// TODO: Implement Shares Split
			})
		})
		list.AddItem("Exact amounts", "Prompt for exact amounts", '7', func() {
			showPromptForm("Exact Amounts", []string{"Amount for " + p1, "Amount for " + p2}, nil, func(vals []string) {
				// TODO: Implement Exact Amounts Split
			})
		})
		list.AddItem("Edit item", "Edit description and cost", 'e', func() {
			showPromptForm("Edit Item", []string{"Description", "Cost"}, []string{item.Description, item.Amount}, func(vals []string) {
				// TODO: Implement Edit Item
			})
		})
		list.AddItem("Delete item", "Remove this item", 'd', func() {
			// TODO: Implement Delete
			closeModal()
		})
		list.AddItem("Duplicate item", "Copy this item", 'c', func() {
			// TODO: Implement Duplicate
			closeModal()
		})
		list.AddItem("Split item", "Split out N items if prefixed with >=2x", 's', func() {
			showPromptForm("Split Item", []string{"Qty to split out (e.g. 1)"}, nil, func(vals []string) {
				// TODO: Implement Split out items
			})
		})
		list.AddItem("Cancel", "", 'q', closeModal)

		list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyLeft {
				closeModal()
				return nil
			}
			return event
		})

		modal := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(list, 25, 1, true).
				AddItem(nil, 0, 1, false), 60, 1, true).
			AddItem(nil, 0, 1, false)

		isModalOpen = true
		pages.AddPage("split_modal", modal, true, true)
		app.SetFocus(list)
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if isModalOpen {
			return event
		}

		// Handle global Tab/Backtab navigation across ALL elements
		if event.Key() == tcell.KeyTab {
			currentFocus := app.GetFocus()
			for i, p := range focusables {
				if p == currentFocus {
					next := (i + 1) % len(focusables)
					app.SetFocus(focusables[next])
					return nil
				}
			}
			app.SetFocus(focusables[0])
			return nil
		} else if event.Key() == tcell.KeyBacktab {
			currentFocus := app.GetFocus()
			for i, p := range focusables {
				if p == currentFocus {
					next := (i - 1 + len(focusables)) % len(focusables)
					app.SetFocus(focusables[next])
					return nil
				}
			}
			app.SetFocus(focusables[len(focusables)-1])
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

		return event
	})

	// Initial focus on first input
	if err := app.SetRoot(pages, true).SetFocus(focusables[0]).Run(); err != nil {
		return err
	}

	return nil
}
