package tui

import (
	"fmt"
	"math"
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

	// Ensure currency defaults to AUD if not set
	if expense.CurrencyCode == "" {
		expense.CurrencyCode = "AUD"

		currentUser, err := splitwise.GetCachedCurrentUser(".cache")
		if err == nil && currentUser.DefaultCurrency != "" {
			expense.CurrencyCode = currentUser.DefaultCurrency
		}
	}

	var p1, p2 string = "P1", "P2"
	for i, eu := range expense.Users {
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))
		if i == 0 {
			p1 = name
		} else if i == 1 {
			p2 = name
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

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true)

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

	itemsTable.SetBorder(true).SetTitle("Items & Splits (Press Enter on an item/paid amount)").SetTitleAlign(tview.AlignLeft)

	pages := tview.NewPages()
	isModalOpen := false
	var focusBeforeModal tview.Primitive

	showPromptForm := func(promptTitle string, fields []string, initialValues []string, onSubmit func(values []string), onCancel func()) {
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
		promptForm.SetBorder(true).SetTitle(promptTitle)

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

	var rowActions map[int]func()
	var refreshItemsTable func()

	refreshItemsTable = func() {
		// Calculate new total based on items, tax, and tip
		var subtotal, taxTotal, tipTotal float64
		for _, item := range parsedDetails.Items {
			cost, _ := strconv.ParseFloat(item.Amount, 64)
			subtotal += cost
		}
		for _, t := range parsedDetails.Tax {
			amt, _ := strconv.ParseFloat(t.Amount, 64)
			taxTotal += amt
		}
		for _, t := range parsedDetails.Tip {
			amt, _ := strconv.ParseFloat(t.Amount, 64)
			tipTotal += amt
		}
		calculatedTotal := subtotal + taxTotal + tipTotal
		formattedCalculatedTotal := fmt.Sprintf("%.2f", calculatedTotal)

		var totalPaid float64
		var paidCounts int
		var onlyPaidIdx int
		for i, eu := range expense.Users {
			paid, _ := strconv.ParseFloat(eu.PaidShare, 64)
			totalPaid += paid
			if paid > 0 {
				paidCounts++
				onlyPaidIdx = i
			}
		}

		formattedTotalPaid := fmt.Sprintf("%.2f", totalPaid)

		// Auto correct logic
		if formattedTotalPaid != formattedCalculatedTotal {
			if math.Abs(totalPaid) < 0.01 && len(expense.Users) > 0 {
				// No one is paying, assign all to first payee
				expense.Users[0].PaidShare = formattedCalculatedTotal
				form.GetFormItemByLabel("Cost").(*tview.InputField).SetText(formattedCalculatedTotal)
				expense.Cost = formattedCalculatedTotal
			} else if paidCounts == 1 {
				// Only 1 person has paid, update them with the new total
				expense.Users[onlyPaidIdx].PaidShare = formattedCalculatedTotal
				form.GetFormItemByLabel("Cost").(*tview.InputField).SetText(formattedCalculatedTotal)
				expense.Cost = formattedCalculatedTotal
			}
			// If neither condition is met, do not auto correct, the mismatch notification will render below
		}

		// Ensure Owed and Net are calculated properly against the newly corrected (or uncorrected) paid shares
		splitwise.CalculateOwed(expense, parsedDetails)

		itemsTable.Clear()
		rowActions = make(map[int]func())

		itemsTable.SetCell(0, 0, tview.NewTableCell("Qty").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter))
		itemsTable.SetCell(0, 1, tview.NewTableCell("Description").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignLeft).SetExpansion(1))
		itemsTable.SetCell(0, 2, tview.NewTableCell("Amount").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignRight))
		itemsTable.SetCell(0, 3, tview.NewTableCell("Shared With (Splits)").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignLeft).SetExpansion(2))

		itemsTable.SetCell(1, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(1, 1, tview.NewTableCell("[ Add Item ]").SetSelectable(true).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		itemsTable.SetCell(1, 2, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(1, 3, tview.NewTableCell("").SetSelectable(false))

		rowActions[1] = func() {
			focusBeforeModal = itemsTable
			isModalOpen = true
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
		}

		row := 2
		var subtotalRender float64
		for i, item := range parsedDetails.Items {
			qty := "1"
			desc := item.Description
			if parts := strings.SplitN(desc, "x ", 2); len(parts) == 2 {
				if _, err := strconv.Atoi(parts[0]); err == nil {
					qty = parts[0]
					desc = parts[1]
				}
			}

			cost, _ := strconv.ParseFloat(item.Amount, 64)
			subtotalRender += cost

			sharedWithStr := ""
			if len(item.SharedWith) > 0 {
				splitAmt := cost / float64(len(item.SharedWith))
				var shares []string
				for _, person := range item.SharedWith {
					shares = append(shares, fmt.Sprintf("%s (%.2f)", person, splitAmt))
				}
				sharedWithStr = strings.Join(shares, ", ")
			}

			itemsTable.SetCell(row, 0, tview.NewTableCell(qty).SetAlign(tview.AlignCenter).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, 1, tview.NewTableCell(desc).SetAlign(tview.AlignLeft).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, 2, tview.NewTableCell(item.Amount).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, 3, tview.NewTableCell(sharedWithStr).SetAlign(tview.AlignLeft).SetTextColor(tcell.ColorWhite))

			rowActions[row] = func(idx int) func() {
				return func() {
					focusBeforeModal = itemsTable
					isModalOpen = true
					itemPtr := &parsedDetails.Items[idx]

					closeModal := func() {
						pages.RemovePage("split_modal")
						isModalOpen = false
						app.SetFocus(focusBeforeModal)
					}

					list := tview.NewList().ShowSecondaryText(true)
					list.SetBorder(true).SetTitle(fmt.Sprintf("Actions: %s", itemPtr.Description))

					list.AddItem(fmt.Sprintf("%s fully pays", p1), "", '1', func() {
						itemPtr.SharedWith = []string{p1}
						refreshItemsTable()
						closeModal()
					})
					list.AddItem(fmt.Sprintf("%s fully pays", p2), "", '2', func() {
						itemPtr.SharedWith = []string{p2}
						refreshItemsTable()
						closeModal()
					})
					list.AddItem("Both half pay for all (50/50)", "", '3', func() {
						itemPtr.SharedWith = []string{p1, p2}
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

					initQty := "1"
					initDesc := itemPtr.Description
					if parts := strings.SplitN(initDesc, "x ", 2); len(parts) == 2 {
						if _, err := strconv.Atoi(parts[0]); err == nil {
							initQty = parts[0]
							initDesc = parts[1]
						}
					}

					list.AddItem("Edit item", "Edit description and cost", 'e', func() {
						showPromptForm("Edit Item", []string{"Qty", "Description", "Cost"}, []string{initQty, initDesc, itemPtr.Amount}, func(vals []string) {
							qty := vals[0]
							desc := vals[1]
							if qty != "1" && qty != "" {
								desc = qty + "x " + desc
							}
							itemPtr.Description = desc
							itemPtr.Amount = vals[2]
							refreshItemsTable()
							closeModal()
						}, func() { app.SetFocus(list) })
					})
					list.AddItem("Delete item", "Remove this item", 'd', func() {
						parsedDetails.Items = append(parsedDetails.Items[:idx], parsedDetails.Items[idx+1:]...)
						refreshItemsTable()
						closeModal()
					})
					list.AddItem("Duplicate item", "Copy this item", 'c', func() {
						newItem := *itemPtr
						parsedDetails.Items = append(parsedDetails.Items[:idx+1], append([]splitwise.Item{newItem}, parsedDetails.Items[idx+1:]...)...)
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
				}
			}(i)
			row++
		}

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("---").SetSelectable(false).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell("---").SetSelectable(false).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Subtotal").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", subtotalRender)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		var taxTotalRender, tipTotalRender float64
		for _, t := range parsedDetails.Tax {
			amt, _ := strconv.ParseFloat(t.Amount, 64)
			taxTotalRender += amt
		}
		for _, t := range parsedDetails.Tip {
			amt, _ := strconv.ParseFloat(t.Amount, 64)
			tipTotalRender += amt
		}

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Tax").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", taxTotalRender)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Tip").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%.2f", tipTotalRender)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		formattedCalculatedTotal = fmt.Sprintf("%.2f", calculatedTotal)
		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Total").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell(formattedCalculatedTotal).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		// Re-evaluate totalPaid after possible autocorrection
		var freshTotalPaid float64
		for _, eu := range expense.Users {
			paid, _ := strconv.ParseFloat(eu.PaidShare, 64)
			freshTotalPaid += paid
		}
		formattedFreshTotalPaid := fmt.Sprintf("%.2f", freshTotalPaid)

		if formattedFreshTotalPaid != formattedCalculatedTotal {
			// Mismatch notification (could not auto-correct cleanly)
			itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
			itemsTable.SetCell(row, 1, tview.NewTableCell("WARNING: Paid != Total").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorRed))
			itemsTable.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%s != %s", formattedFreshTotalPaid, formattedCalculatedTotal)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorRed))
			itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
			row++
		}

		// Add "Amounts Paid" directly into the items table
		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("--- Amounts Paid ---").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		for i, eu := range expense.Users {
			lastName := ""
			if eu.User.LastName != nil {
				lastName = *eu.User.LastName
			}
			name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))

			itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
			itemsTable.SetCell(row, 1, tview.NewTableCell(name).SetSelectable(true).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, 2, tview.NewTableCell(eu.PaidShare).SetSelectable(true).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))

			rowActions[row] = func(idx int, userName string) func() {
				return func() {
					focusBeforeModal = itemsTable
					isModalOpen = true
					euPtr := &expense.Users[idx]

					showPromptForm(fmt.Sprintf("Edit Paid Amount: %s", userName), []string{"Paid Amount"}, []string{euPtr.PaidShare}, func(vals []string) {
						euPtr.PaidShare = vals[0]

						// Update the main Cost form input field to match total sum of new custom payments
						var newTotalPaid float64
						for _, u := range expense.Users {
							p, _ := strconv.ParseFloat(u.PaidShare, 64)
							newTotalPaid += p
						}
						expense.Cost = fmt.Sprintf("%.2f", newTotalPaid)
						form.GetFormItemByLabel("Cost").(*tview.InputField).SetText(expense.Cost)

						refreshItemsTable()
						isModalOpen = false
					}, func() {
						isModalOpen = false
						app.SetFocus(itemsTable)
					})
				}
			}(i, name)
			row++
		}

		// Add "Amounts Owed" Breakdown
		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("--- Amounts Owed ---").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 3, tview.NewTableCell("").SetSelectable(false))
		row++

		for _, eu := range expense.Users {
			lastName := ""
			if eu.User.LastName != nil {
				lastName = *eu.User.LastName
			}
			name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))

			net, _ := strconv.ParseFloat(eu.NetBalance, 64)
			balanceText := ""
			if net > 0 {
				balanceText = fmt.Sprintf("(Is Owed: %.2f)", net)
			} else if net < 0 {
				balanceText = fmt.Sprintf("(Owes: %.2f)", -net)
			} else {
				balanceText = "(Settled)"
			}

			itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
			itemsTable.SetCell(row, 1, tview.NewTableCell(name).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
			itemsTable.SetCell(row, 2, tview.NewTableCell(eu.OwedShare).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
			itemsTable.SetCell(row, 3, tview.NewTableCell(balanceText).SetSelectable(false).SetAlign(tview.AlignLeft).SetTextColor(tcell.ColorGreen))
			row++
		}
	}

	refreshItemsTable()

	itemsTable.SetSelectedFunc(func(row, column int) {
		if action, ok := rowActions[row]; ok {
			action()
		}
	})

	// Focus management array - gather all focusable elements directly
	var focusables []tview.Primitive
	for i := 0; i < form.GetFormItemCount(); i++ {
		focusables = append(focusables, form.GetFormItem(i))
	}
	focusables = append(focusables, itemsTable)
	for i := 0; i < actionButtons.GetButtonCount(); i++ {
		focusables = append(focusables, actionButtons.GetButton(i))
	}

	// Help Text
	helpText := `Keyboard Shortcuts:
[?] Toggle Help      | [Tab]/[Backtab] Switch Focus
(Press Enter on an item or paid amount to edit)
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

	pages = tview.NewPages()
	pages.AddPage("main", rootFlex, true, true)

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
