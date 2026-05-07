package tui

import (
	"fmt"
	"strconv"
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
	if parsedDetails == nil {
		parsedDetails = &splitwise.ItemizedDetail{}
	}
	notesText := parsedDetails.Notes

	// Ensure currency defaults to AUD if not set, though it should be pulled from User
	if expense.CurrencyCode == "" {
		expense.CurrencyCode = "AUD"

		currentUser, err := splitwise.GetCachedCurrentUser(".cache")
		if err == nil && currentUser.DefaultCurrency != "" {
			expense.CurrencyCode = currentUser.DefaultCurrency
		}
	}

	// Basic Info Form
	form := tview.NewForm().
		AddInputField("Description", expense.Description, 40, nil, func(text string) {
			expense.Description = text
		}).
		AddInputField("Cost", expense.Cost, 20, nil, func(text string) {
			expense.Cost = text
		}).
		AddInputField("Currency", expense.CurrencyCode, 10, nil, func(text string) {
			expense.CurrencyCode = text
		}).
		AddInputField("Date", expense.Date, 25, nil, func(text string) {
			expense.Date = text
		}).
		AddTextArea("Notes", notesText, 40, 5, 0, func(text string) {
			parsedDetails.Notes = text
		})

	form.SetBorder(true).SetTitle("Basic Info").SetTitleAlign(tview.AlignLeft)

	// Split amounts table (Paid/Owed balances) - Moved to left pane
	splitTable := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	splitTable.SetBorder(true).SetTitle("Amounts Paid").SetTitleAlign(tview.AlignLeft)

	splitTable.SetCell(0, 0, tview.NewTableCell("User").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 1, tview.NewTableCell("Paid").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))
	splitTable.SetCell(0, 2, tview.NewTableCell("Owed").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter).SetExpansion(1))

	var p1, p2 string = "P1", "P2"
	userMap := make(map[string]*splitwise.ExpenseUser)
	for i, eu := range expense.Users {
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))
		userMap[name] = &expense.Users[i]

		if i == 0 {
			p1 = name
		} else if i == 1 {
			p2 = name
		}
	}

	refreshSplitTable := func() {
		// Recalculate owed amounts
		for _, eu := range expense.Users {
			eu.OwedShare = "0.00"
		}

		for _, item := range parsedDetails.Items {
			cost, _ := strconv.ParseFloat(item.Amount, 64)
			splitAmt := cost / float64(len(item.SharedWith))
			for _, personName := range item.SharedWith {
				if user, ok := userMap[personName]; ok {
					currentOwed, _ := strconv.ParseFloat(user.OwedShare, 64)
					user.OwedShare = fmt.Sprintf("%.2f", currentOwed+splitAmt)
				}
			}
		}

		for _, item := range parsedDetails.Tax {
			if user, ok := userMap[item.Name]; ok {
				cost, _ := strconv.ParseFloat(item.Amount, 64)
				currentOwed, _ := strconv.ParseFloat(user.OwedShare, 64)
				user.OwedShare = fmt.Sprintf("%.2f", currentOwed+cost)
			}
		}

		for _, item := range parsedDetails.Tip {
			if user, ok := userMap[item.Name]; ok {
				cost, _ := strconv.ParseFloat(item.Amount, 64)
				currentOwed, _ := strconv.ParseFloat(user.OwedShare, 64)
				user.OwedShare = fmt.Sprintf("%.2f", currentOwed+cost)
			}
		}

		// Update table
		for row, eu := range expense.Users {
			lastName := ""
			if eu.User.LastName != nil {
				lastName = *eu.User.LastName
			}
			name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))

			paid, _ := strconv.ParseFloat(eu.PaidShare, 64)
			owed, _ := strconv.ParseFloat(eu.OwedShare, 64)
			net := paid - owed

			splitTable.SetCell(row+1, 0, tview.NewTableCell(name).SetAlign(tview.AlignLeft))
			splitTable.SetCell(row+1, 1, tview.NewTableCell(eu.PaidShare).SetAlign(tview.AlignRight))
			splitTable.SetCell(row+1, 2, tview.NewTableCell(fmt.Sprintf("%.2f", owed)).SetAlign(tview.AlignRight))
			splitTable.SetCell(row+1, 3, tview.NewTableCell(fmt.Sprintf("%.2f", net)).SetAlign(tview.AlignRight))
		}
	}

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 2, true).
		AddItem(splitTable, 0, 1, false)

	// Action Buttons at the bottom
	actionButtons := tview.NewForm()
	actionButtons.AddButton("Save (Not Implemented)", func() {
		app.Stop()
	}).
		AddButton("Quit", func() {
			app.Stop()
		})
	actionButtons.SetButtonsAlign(tview.AlignCenter)

	// Items Table
	itemsTable := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	itemsTable.SetBorder(true).SetTitle("Items & Splits (Press Enter on an item)").SetTitleAlign(tview.AlignLeft)

	refreshItemsTable := func() {
		itemsTable.Clear()
		itemsTable.SetCell(0, 0, tview.NewTableCell("Qty").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
		itemsTable.SetCell(0, 1, tview.NewTableCell("Description").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignLeft).SetExpansion(1))
		itemsTable.SetCell(0, 2, tview.NewTableCell("Amount").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignRight))
		itemsTable.SetCell(0, 3, tview.NewTableCell("Shared With (Splits)").SetSelectable(false).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignLeft).SetExpansion(2))

		itemsTable.SetCell(1, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(1, 1, tview.NewTableCell("[ Add Item ]").SetSelectable(true).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignLeft))
		itemsTable.SetCell(1, 2, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(1, 3, tview.NewTableCell("").SetSelectable(false))

		row := 2
		var subtotal float64
		for _, item := range parsedDetails.Items {
			qty := "1"
			desc := item.Description
			if parts := strings.SplitN(desc, "x ", 2); len(parts) == 2 {
				if _, err := strconv.Atoi(parts[0]); err == nil {
					qty = parts[0]
					desc = parts[1]
				}
			}

			cost, _ := strconv.ParseFloat(item.Amount, 64)
			subtotal += cost

			sharedWithStr := ""
			if len(item.SharedWith) > 0 {
				splitAmt := cost / float64(len(item.SharedWith))
				var shares []string
				for _, person := range item.SharedWith {
					shares = append(shares, fmt.Sprintf("%s (%.2f)", person, splitAmt))
				}
				sharedWithStr = strings.Join(shares, ", ")
			}

			itemsTable.SetCell(row, 0, tview.NewTableCell(qty).SetAlign(tview.AlignCenter))
			itemsTable.SetCell(row, 1, tview.NewTableCell(desc).SetAlign(tview.AlignLeft))
			itemsTable.SetCell(row, 2, tview.NewTableCell(item.Amount).SetAlign(tview.AlignRight))
			itemsTable.SetCell(row, 3, tview.NewTableCell(sharedWithStr).SetAlign(tview.AlignLeft))
			row++
		}

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("---").SetSelectable(false))
		itemsTable.SetCell(row, 2, tview.NewTableCell("---").SetSelectable(false))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Subtotal").SetSelectable(false).SetAlign(tview.AlignRight))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", subtotal)).SetSelectable(false).SetAlign(tview.AlignRight))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		var taxTotal, tipTotal float64
		for _, t := range parsedDetails.Tax {
			amt, _ := strconv.ParseFloat(t.Amount, 64)
			taxTotal += amt
		}
		for _, t := range parsedDetails.Tip {
			amt, _ := strconv.ParseFloat(t.Amount, 64)
			tipTotal += amt
		}

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Tax").SetSelectable(false).SetAlign(tview.AlignRight))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", taxTotal)).SetSelectable(false).SetAlign(tview.AlignRight))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Tip").SetSelectable(false).SetAlign(tview.AlignRight))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", tipTotal)).SetSelectable(false).SetAlign(tview.AlignRight))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Total").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorYellow))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", subtotal+taxTotal+tipTotal)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorYellow))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))

		refreshSplitTable()
	}

	refreshItemsTable()

	// Focus management array - gather all focusable elements directly
	var focusables []tview.Primitive
	for i := 0; i < form.GetFormItemCount(); i++ {
		focusables = append(focusables, form.GetFormItem(i))
	}
	focusables = append(focusables, splitTable)
	focusables = append(focusables, itemsTable)
	for i := 0; i < actionButtons.GetButtonCount(); i++ {
		focusables = append(focusables, actionButtons.GetButton(i))
	}

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
	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(leftFlex, 0, 1, true).
		AddItem(itemsTable, 0, 2, false)

	rootFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 3, 1, false).
		AddItem(mainFlex, 0, 1, true).
		AddItem(actionButtons, 3, 1, false)

	pages := tview.NewPages()
	pages.AddPage("main", rootFlex, true, true)

	isModalOpen := false
	var focusBeforeModal tview.Primitive

	showPromptForm := func(title string, fields []string, initialValues []string, onSubmit func(values []string), onCancel func()) {
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

			// Set focus back to what it was
			app.SetFocus(focusBeforeModal)
		})
		promptForm.AddButton("Cancel", func() {
			pages.RemovePage("prompt_modal")
			onCancel()
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

	// Split table action handling
	splitTable.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return
		}

		userIdx := row - 1
		if userIdx < 0 || userIdx >= len(expense.Users) {
			return
		}
		eu := &expense.Users[userIdx]

		name := splitTable.GetCell(row, 0).Text

		focusBeforeModal = splitTable
		isModalOpen = true

		showPromptForm(fmt.Sprintf("Edit Paid Amount: %s", name), []string{"Paid Amount"}, []string{eu.PaidShare}, func(vals []string) {
			eu.PaidShare = vals[0]
			refreshItemsTable()
			isModalOpen = false
		}, func() {
			isModalOpen = false
			app.SetFocus(splitTable)
		})
	})

	// Items table action handling
	itemsTable.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return
		}

		focusBeforeModal = itemsTable
		isModalOpen = true

		closeModal := func() {
			pages.RemovePage("split_modal")
			isModalOpen = false
			app.SetFocus(focusBeforeModal)
		}

		if row == 1 { // Add Item
			showPromptForm("Add Item", []string{"Qty", "Description", "Cost"}, []string{"1", "", "0.00"}, func(vals []string) {
				qty := vals[0]
				desc := vals[1]
				if qty != "1" && qty != "" {
					desc = qty + "x " + desc
				}
				parsedDetails.Items = append(parsedDetails.Items, splitwise.Item{
					Description: desc,
					Amount:      vals[2],
					SharedWith:  []string{p1, p2}, // Default to both
				})
				refreshItemsTable()
				isModalOpen = false
			}, func() {
				isModalOpen = false
				app.SetFocus(itemsTable)
			})
			return
		}

		itemIdx := row - 2
		if itemIdx < 0 || itemIdx >= len(parsedDetails.Items) {
			isModalOpen = false
			return
		}
		item := &parsedDetails.Items[itemIdx]

		list := tview.NewList().ShowSecondaryText(true)
		list.SetBorder(true).SetTitle(fmt.Sprintf("Actions: %s", item.Description))

		list.AddItem(fmt.Sprintf("%s fully pays", p1), "", '1', func() {
			item.SharedWith = []string{p1}
			refreshItemsTable()
			closeModal()
		})
		list.AddItem(fmt.Sprintf("%s fully pays", p2), "", '2', func() {
			item.SharedWith = []string{p2}
			refreshItemsTable()
			closeModal()
		})
		list.AddItem("Both half pay for all (50/50)", "", '3', func() {
			item.SharedWith = []string{p1, p2}
			refreshItemsTable()
			closeModal()
		})
		list.AddItem(fmt.Sprintf("%s pays N items, %s pays remaining", p1, p2), "Qty split based on '1x ' prefix", '4', func() {
			showPromptForm("Split N Items", []string{"Items for " + p1, "Items for " + p2}, nil, func(vals []string) {
				// Stub
				closeModal()
			}, func() { app.SetFocus(list) })
		})
		list.AddItem("Different %", "Prompt for percentage", '5', func() {
			showPromptForm("Percentage Split", []string{"% for " + p1, "% for " + p2}, nil, func(vals []string) {
				// Stub
				closeModal()
			}, func() { app.SetFocus(list) })
		})
		list.AddItem("Shares", "Prompt for shares", '6', func() {
			showPromptForm("Shares Split", []string{"Shares for " + p1, "Shares for " + p2}, nil, func(vals []string) {
				// Stub
				closeModal()
			}, func() { app.SetFocus(list) })
		})
		list.AddItem("Exact amounts", "Prompt for exact amounts", '7', func() {
			showPromptForm("Exact Amounts", []string{"Amount for " + p1, "Amount for " + p2}, nil, func(vals []string) {
				// Stub
				closeModal()
			}, func() { app.SetFocus(list) })
		})

		// Parse out initial qty and desc for editing
		initQty := "1"
		initDesc := item.Description
		if parts := strings.SplitN(initDesc, "x ", 2); len(parts) == 2 {
			if _, err := strconv.Atoi(parts[0]); err == nil {
				initQty = parts[0]
				initDesc = parts[1]
			}
		}

		list.AddItem("Edit item", "Edit description and cost", 'e', func() {
			showPromptForm("Edit Item", []string{"Qty", "Description", "Cost"}, []string{initQty, initDesc, item.Amount}, func(vals []string) {
				qty := vals[0]
				desc := vals[1]
				if qty != "1" && qty != "" {
					desc = qty + "x " + desc
				}
				item.Description = desc
				item.Amount = vals[2]
				refreshItemsTable()
				closeModal()
			}, func() { app.SetFocus(list) })
		})
		list.AddItem("Delete item", "Remove this item", 'd', func() {
			parsedDetails.Items = append(parsedDetails.Items[:itemIdx], parsedDetails.Items[itemIdx+1:]...)
			refreshItemsTable()
			closeModal()
		})
		list.AddItem("Duplicate item", "Copy this item", 'c', func() {
			newItem := *item
			parsedDetails.Items = append(parsedDetails.Items[:itemIdx+1], append([]splitwise.Item{newItem}, parsedDetails.Items[itemIdx+1:]...)...)
			refreshItemsTable()
			closeModal()
		})
		list.AddItem("Split item", "Split out N items if prefixed with >=2x", 's', func() {
			showPromptForm("Split Item", []string{"Qty to split out (e.g. 1)"}, nil, func(vals []string) {
				// Stub
				closeModal()
			}, func() { app.SetFocus(list) })
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

		pages.AddPage("split_modal", modal, true, true)
		app.SetFocus(list)
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if isModalOpen {
			return event
		}

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

	if err := app.SetRoot(pages, true).SetFocus(focusables[0]).Run(); err != nil {
		return err
	}

	return nil
}
