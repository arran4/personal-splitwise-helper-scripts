package tui

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func splitTUIDescriptionExtra(description string) (string, string) {
	parts := strings.SplitN(description, " | ", 2)
	if len(parts) == 1 {
		return description, ""
	}
	return parts[0], parts[1]
}

func isFiniteNumber(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func EditExpense(expense *splitwise.DetailedExpense, opts ...EditExpenseOption) (bool, []byte, error) {
	config := &editExpenseConfig{}
	for _, opt := range opts {
		opt(config)
	}

	app := tview.NewApplication()
	sent := false
	var sendResponse []byte

	// Title
	title := tview.NewTextView().
		SetText(fmt.Sprintf("Editing Expense: [yellow]%s[white] (ID: %d)", expense.Description, expense.ID)).
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
		AddInputField("Description", expense.Description, 0, nil, func(text string) {
			expense.Description = text
		}).
		AddInputField("Cost", expense.Cost, 0, nil, func(text string) {
			expense.Cost = text
		}).
		AddInputField("Currency", expense.CurrencyCode, 0, nil, func(text string) {
			expense.CurrencyCode = text
		}).
		AddInputField("Date", expense.Date, 0, nil, func(text string) {
			expense.Date = text
		}).
		AddTextArea("Notes", notesText, 0, 5, 0, func(text string) {
			parsedDetails.Notes = text
		})

	form.SetBorder(true).SetTitle("Basic Info").SetTitleAlign(tview.AlignLeft)

	leftFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true)

	// Items Table
	itemsTable := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSelectedStyle(tcell.StyleDefault)

	itemsTable.SetBorder(true).SetTitle("Items & Splits (Press Enter on an item/paid amount)").SetTitleAlign(tview.AlignLeft)
	focusedItemsTableStyle := tcell.StyleDefault
	unfocusedItemsTableStyle := tcell.StyleDefault.Foreground(tview.Styles.PrimaryTextColor).Background(tview.Styles.PrimitiveBackgroundColor)
	setItemsTableFocused := func(focused bool) {
		if focused {
			itemsTable.SetSelectedStyle(focusedItemsTableStyle)
			return
		}
		itemsTable.SetSelectedStyle(unfocusedItemsTableStyle)
	}
	setItemsTableFocused(false)
	setFocus := func(p tview.Primitive) {
		setItemsTableFocused(p == itemsTable)
		app.SetFocus(p)
	}

	var pages *tview.Pages // Initialize below
	isModalOpen := false
	var focusBeforeModal tview.Primitive
	showMessageModal := func(title, message string) {
		previousFocus := app.GetFocus()
		isModalOpen = true
		modal := tview.NewModal().
			SetText(message).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(_ int, _ string) {
				pages.RemovePage("message_modal")
				isModalOpen = false
				setFocus(previousFocus)
			})
		modal.SetTitle(title)
		pages.AddPage("message_modal", modal, true, true)
		setFocus(modal)
	}
	showTextModal := func(title, content string) {
		previousFocus := app.GetFocus()
		isModalOpen = true
		textView := tview.NewTextView().
			SetText(content).
			SetDynamicColors(true).
			SetScrollable(true).
			SetWrap(false)
		textView.SetBorder(true).SetTitle(title)
		closeModal := func() {
			pages.RemovePage("text_modal")
			isModalOpen = false
			setFocus(previousFocus)
		}
		textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
				closeModal()
				return nil
			}
			return event
		})
		modal := tview.NewFlex().
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(textView, 0, 1, true).
				AddItem(nil, 0, 0, false), 0, 1, true)
		pages.AddPage("text_modal", modal, true, true)
		setFocus(textView)
	}

	showPromptForm := func(promptTitle string, fields []string, initialValues []string, onSubmit func(values []string) bool, onCancel func()) {
		promptForm := tview.NewForm()
		for i, f := range fields {
			initVal := ""
			if i < len(initialValues) {
				initVal = initialValues[i]
			}
			promptForm.AddInputField(f, initVal, 0, nil, nil)
		}
		promptForm.AddButton("Save", func() {
			var vals []string
			for i := 0; i < len(fields); i++ {
				vals = append(vals, promptForm.GetFormItem(i).(*tview.InputField).GetText())
			}
			if !onSubmit(vals) {
				return
			}
			pages.RemovePage("prompt_modal")

			// Set focus back to what it was
			setFocus(focusBeforeModal)
		})
		promptForm.AddButton("Cancel", func() {
			pages.RemovePage("prompt_modal")
			onCancel()
		})
		promptForm.SetBorder(true).SetTitle(promptTitle)
		for i, field := range fields {
			if field == "Description" {
				promptForm.SetFocus(i)
				break
			}
		}

		modalForm := tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(promptForm, 15, 1, true).
				AddItem(nil, 0, 1, false), 60, 1, true).
			AddItem(nil, 0, 1, false)

		pages.AddPage("prompt_modal", modalForm, true, true)
		setFocus(promptForm)
	}

	var rowActions map[int]func()
	var refreshItemsTable func()
	calculateDetailsTotal := func() float64 {
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
		return subtotal + taxTotal + tipTotal
	}
	lastCalculatedTotal := calculateDetailsTotal()
	hasQuantityItems := func() bool {
		for _, item := range parsedDetails.Items {
			qty, _ := splitwise.ParseItemDescription(item.Description)
			if qty > 1 {
				return true
			}
		}
		return false
	}
	formatMoney := func(v float64) string {
		return fmt.Sprintf("%.2f", v)
	}
	buildWeightedSharedWith := func(p1Weight, p2Weight int) []string {
		sharedWith := make([]string, 0, p1Weight+p2Weight)
		for i := 0; i < p1Weight; i++ {
			sharedWith = append(sharedWith, p1)
		}
		for i := 0; i < p2Weight; i++ {
			sharedWith = append(sharedWith, p2)
		}
		return sharedWith
	}
	itemParticipantAmounts := func(item splitwise.Item) (float64, float64, int, int, int) {
		totalAmount, _ := strconv.ParseFloat(item.Amount, 64)
		p1Weight := 0
		p2Weight := 0
		for _, person := range item.SharedWith {
			switch person {
			case p1:
				p1Weight++
			case p2:
				p2Weight++
			}
		}
		totalWeight := p1Weight + p2Weight
		if totalWeight == 0 {
			return 0, 0, 0, 0, 0
		}
		p1Amount := totalAmount * float64(p1Weight) / float64(totalWeight)
		p2Amount := totalAmount - p1Amount
		return p1Amount, p2Amount, p1Weight, p2Weight, totalWeight
	}
	reduceRatio := func(a, b int) (int, int) {
		if a == 0 || b == 0 {
			return a, b
		}
		gcd := func(x, y int) int {
			for y != 0 {
				x, y = y, x%y
			}
			return x
		}
		d := gcd(a, b)
		return a / d, b / d
	}
	buildCurrentExpenseState := func() splitwise.DetailedExpense {
		current := *expense
		current.Users = append([]splitwise.ExpenseUser(nil), expense.Users...)
		current.Details = splitwise.SerializeDetails(parsedDetails)
		return current
	}
	currentStateJSON := func() ([]byte, error) {
		current := buildCurrentExpenseState()
		return json.MarshalIndent(current, "", "  ")
	}
	receiptImageURLs := func() []string {
		seen := make(map[string]bool)
		var urls []string
		collect := func(v interface{}) {
			if s, ok := v.(string); ok && s != "" && !seen[s] {
				seen[s] = true
				urls = append(urls, s)
			}
		}
		collect(expense.Receipt.Original)
		collect(expense.Receipt.Large)
		return urls
	}
	openURL := func(target string) error {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", target)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
		default:
			cmd = exec.Command("xdg-open", target)
		}
		return cmd.Start()
	}

	// Action Buttons at the bottom
	actionButtons := tview.NewForm()
	actionButtons.AddButton("Send", func() {
		focusBeforeModal = actionButtons
		current := buildCurrentExpenseState()
		client, err := splitwise.NewClient()
		if err != nil {
			showMessageModal("Send Error", err.Error())
			return
		}
		var response []byte
		if current.ID == 0 {
			response, err = client.CreateExpense(&current)
		} else {
			response, err = client.UpdateExpense(&current)
		}
		if err != nil {
			showMessageModal("Send Error", err.Error())
			return
		}
		*expense = current
		sent = true
		sendResponse = response
		app.Stop()
	}).AddButton("Quit", func() {
		app.Stop()
	})
	if len(receiptImageURLs()) > 0 {
		actionButtons.AddButton("View Image(s)", func() {
			focusBeforeModal = actionButtons
			urls := receiptImageURLs()
			for _, u := range urls {
				if err := openURL(u); err != nil {
					showMessageModal("Image Error", err.Error())
					return
				}
			}
		})
	}
	actionButtons.AddButton("View Raw JSON", func() {
		focusBeforeModal = actionButtons
		data, err := currentStateJSON()
		if err != nil {
			showMessageModal("JSON Error", err.Error())
			return
		}
		showTextModal("Current State JSON", string(data))
	}).AddButton("Export Raw JSON", func() {
		focusBeforeModal = actionButtons
		defaultPath := fmt.Sprintf("expense_%d_current.json", expense.ID)
		showPromptForm("Export Raw JSON", []string{"File Path"}, []string{defaultPath}, func(vals []string) bool {
			target := strings.TrimSpace(vals[0])
			if target == "" {
				return false
			}
			data, err := currentStateJSON()
			if err != nil {
				showMessageModal("Export Error", err.Error())
				return false
			}
			if err := os.WriteFile(target, data, 0644); err != nil {
				showMessageModal("Export Error", err.Error())
				return false
			}
			showMessageModal("Exported", fmt.Sprintf("Wrote current state to %s", target))
			return true
		}, func() {
			setFocus(actionButtons)
		})
	})
	actionButtons.SetButtonsAlign(tview.AlignCenter)

	refreshItemsTable = func() {
		calculatedTotal := calculateDetailsTotal()
		formattedCalculatedTotal := fmt.Sprintf("%.2f", calculatedTotal)
		showPerItemColumn := hasQuantityItems()

		if fmt.Sprintf("%.2f", lastCalculatedTotal) != formattedCalculatedTotal {
			splitwise.AutoCorrectPaidShares(expense, lastCalculatedTotal, calculatedTotal)
			expense.Cost = formattedCalculatedTotal
			form.GetFormItemByLabel("Cost").(*tview.InputField).SetText(formattedCalculatedTotal)
			lastCalculatedTotal = calculatedTotal
		}

		// Ensure Owed and Net are calculated properly against the newly corrected (or uncorrected) paid shares
		splitwise.CalculateOwed(expense, parsedDetails)

		itemsTable.Clear()
		rowActions = make(map[int]func())

		itemsTable.SetCell(0, 0, tview.NewTableCell("Qty").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignCenter))
		itemsTable.SetCell(0, 1, tview.NewTableCell("Description").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignLeft).SetExpansion(1))
		totalCol := 2
		sharedCol := 3
		if showPerItemColumn {
			itemsTable.SetCell(0, 2, tview.NewTableCell("Per Item").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignRight))
			totalCol = 3
			sharedCol = 4
		}
		itemsTable.SetCell(0, totalCol, tview.NewTableCell("Total").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignRight))
		itemsTable.SetCell(0, sharedCol, tview.NewTableCell("Shared With (Splits)").SetSelectable(false).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignLeft).SetExpansion(2))

		itemsTable.SetCell(1, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(1, 1, tview.NewTableCell("[ Add Item ]").SetSelectable(true).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		if showPerItemColumn {
			itemsTable.SetCell(1, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(1, totalCol, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(1, sharedCol, tview.NewTableCell("").SetSelectable(false))

		rowActions[1] = func() {
			focusBeforeModal = itemsTable
			isModalOpen = true
			showPromptForm("Add Item", []string{"Qty", "Description", "Per Item Cost"}, []string{"1", "", "0.00"}, func(vals []string) bool {
				qty := 1
				if parsedQty, err := strconv.Atoi(strings.TrimSpace(vals[0])); err == nil && parsedQty > 0 {
					qty = parsedQty
				}
				desc := splitwise.FormatItemDescription(qty, vals[1])
				perItemCost, err := strconv.ParseFloat(strings.TrimSpace(vals[2]), 64)
				if err != nil || !isFiniteNumber(perItemCost) || perItemCost < 0 {
					setFocus(itemsTable)
					return false
				}
				parsedDetails.Items = append(parsedDetails.Items, splitwise.Item{
					Description: desc,
					Amount:      formatMoney(perItemCost * float64(qty)),
					SharedWith:  []string{p1, p2}, // Default to both
				})
				refreshItemsTable()
				isModalOpen = false
				return true
			}, func() {
				isModalOpen = false
				setFocus(itemsTable)
			})
		}

		row := 2
		var subtotalRender float64
		for i, item := range parsedDetails.Items {
			qtyInt, desc := splitwise.ParseItemDescription(item.Description)
			if qtyInt <= 0 {
				qtyInt = 1
			}
			displayDesc, _ := splitTUIDescriptionExtra(desc)
			qty := strconv.Itoa(qtyInt)

			cost, _ := strconv.ParseFloat(item.Amount, 64)
			subtotalRender += cost
			perItemCost := cost / float64(qtyInt)

			sharedWithStr := ""
			if len(item.SharedWith) > 0 {
				splitAmt := cost / float64(len(item.SharedWith))
				counts := make(map[string]int)
				var orderedPeople []string
				for _, person := range item.SharedWith {
					if counts[person] == 0 {
						orderedPeople = append(orderedPeople, person)
					}
					counts[person]++
				}

				var shares []string
				for _, person := range orderedPeople {
					count := counts[person]
					totalShare := splitAmt * float64(count)
					if count > 1 && len(item.SharedWith) == qtyInt {
						shares = append(shares, fmt.Sprintf("%s (%.2f, %d items)", person, totalShare, count))
						continue
					}
					shares = append(shares, fmt.Sprintf("%s (%.2f)", person, totalShare))
				}
				sharedWithStr = strings.Join(shares, ", ")
			}

			itemsTable.SetCell(row, 0, tview.NewTableCell(qty).SetSelectable(true).SetAlign(tview.AlignCenter).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, 1, tview.NewTableCell(displayDesc).SetSelectable(true).SetAlign(tview.AlignLeft).SetTextColor(tcell.ColorWhite))
			if showPerItemColumn {
				itemsTable.SetCell(row, 2, tview.NewTableCell(formatMoney(perItemCost)).SetSelectable(true).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			}
			itemsTable.SetCell(row, totalCol, tview.NewTableCell(item.Amount).SetSelectable(true).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, sharedCol, tview.NewTableCell(sharedWithStr).SetSelectable(true).SetAlign(tview.AlignLeft).SetTextColor(tcell.ColorWhite))

			rowActions[row] = func(idx int) func() {
				return func() {
					focusBeforeModal = itemsTable
					isModalOpen = true
					itemPtr := &parsedDetails.Items[idx]

					closeModal := func() {
						pages.RemovePage("split_modal")
						isModalOpen = false
						setFocus(focusBeforeModal)
					}

					list := tview.NewList().ShowSecondaryText(true)
					itemQty, itemDesc := splitwise.ParseItemDescription(itemPtr.Description)
					actionDesc, _ := splitTUIDescriptionExtra(itemDesc)
					list.SetBorder(true).SetTitle(fmt.Sprintf("Actions: %s", splitwise.FormatItemDescription(itemQty, actionDesc)))

					list.AddItem(fmt.Sprintf("%s owes full item", p1), "", '1', func() {
						itemPtr.SharedWith = []string{p1}
						refreshItemsTable()
						closeModal()
					})
					list.AddItem(fmt.Sprintf("%s owes full item", p2), "", '2', func() {
						itemPtr.SharedWith = []string{p2}
						refreshItemsTable()
						closeModal()
					})
					list.AddItem("Both owe half (50/50)", "", '3', func() {
						itemPtr.SharedWith = []string{p1, p2}
						refreshItemsTable()
						closeModal()
					})
					list.AddItem(fmt.Sprintf("%s owes N items, %s owes remaining", p1, p2), "Uses item quantity; plain items count as 1", '4', func() {
						showPromptForm("Split N Items", []string{"Items owed by " + p1, "Items owed by " + p2}, []string{"", ""}, func(vals []string) bool {
							totalQty, _ := splitwise.ParseItemDescription(itemPtr.Description)

							p1Raw := strings.TrimSpace(vals[0])
							p2Raw := strings.TrimSpace(vals[1])
							if p1Raw == "" && p2Raw == "" {
								setFocus(list)
								return false
							}

							p1Qty := 0
							p2Qty := 0
							if p1Raw != "" {
								parsed, err := strconv.Atoi(p1Raw)
								if err != nil || parsed < 0 {
									setFocus(list)
									return false
								}
								p1Qty = parsed
							}
							if p2Raw != "" {
								parsed, err := strconv.Atoi(p2Raw)
								if err != nil || parsed < 0 {
									setFocus(list)
									return false
								}
								p2Qty = parsed
							}

							switch {
							case p1Raw == "":
								p1Qty = totalQty - p2Qty
							case p2Raw == "":
								p2Qty = totalQty - p1Qty
							}

							if p1Qty < 0 || p2Qty < 0 || p1Qty+p2Qty != totalQty {
								setFocus(list)
								return false
							}

							sharedWith := make([]string, 0, totalQty)
							for i := 0; i < p1Qty; i++ {
								sharedWith = append(sharedWith, p1)
							}
							for i := 0; i < p2Qty; i++ {
								sharedWith = append(sharedWith, p2)
							}

							itemPtr.SharedWith = sharedWith
							refreshItemsTable()
							closeModal()
							return true
						}, func() { setFocus(list) })
					})
					list.AddItem("Different % owed", "Prompt for percentage", '5', func() {
						p1Amount, _, _, _, totalWeight := itemParticipantAmounts(*itemPtr)
						totalAmount, _ := strconv.ParseFloat(itemPtr.Amount, 64)
						p1InitialPct := "0.00"
						p2InitialPct := "0.00"
						if totalAmount > 0 && totalWeight > 0 {
							p1Pct := math.Round((p1Amount/totalAmount)*10000) / 100
							p2Pct := math.Round((100-p1Pct)*100) / 100
							p1InitialPct = fmt.Sprintf("%.2f", p1Pct)
							p2InitialPct = fmt.Sprintf("%.2f", p2Pct)
						}
						showPromptForm("Percentage Split", []string{"% owed by " + p1, "% owed by " + p2}, []string{p1InitialPct, p2InitialPct}, func(vals []string) bool {
							p1Raw := strings.TrimSpace(vals[0])
							p2Raw := strings.TrimSpace(vals[1])
							if p1Raw == "" && p2Raw == "" {
								setFocus(list)
								return false
							}

							p1Pct := 0.0
							p2Pct := 0.0
							if p1Raw != "" {
								parsed, err := strconv.ParseFloat(p1Raw, 64)
								if err != nil || !isFiniteNumber(parsed) || parsed < 0 {
									setFocus(list)
									return false
								}
								p1Pct = parsed
							}
							if p2Raw != "" {
								parsed, err := strconv.ParseFloat(p2Raw, 64)
								if err != nil || !isFiniteNumber(parsed) || parsed < 0 {
									setFocus(list)
									return false
								}
								p2Pct = parsed
							}

							switch {
							case p1Raw == "":
								p1Pct = 100 - p2Pct
							case p2Raw == "":
								p2Pct = 100 - p1Pct
							}

							if p1Pct < 0 || p2Pct < 0 || math.Abs((p1Pct+p2Pct)-100) > 0.0001 {
								setFocus(list)
								return false
							}

							totalAmount, err := strconv.ParseFloat(itemPtr.Amount, 64)
							if err != nil || !isFiniteNumber(totalAmount) || totalAmount < 0 {
								setFocus(list)
								return false
							}
							totalCents := int(math.Round(totalAmount * 100))
							p1Cents := int(math.Round((p1Pct / 100) * float64(totalCents)))
							p2Cents := totalCents - p1Cents
							itemPtr.SharedWith = buildWeightedSharedWith(p1Cents, p2Cents)
							closeModal()
							refreshItemsTable()
							return true
						}, func() { setFocus(list) })
					})
					list.AddItem("Shares owed", "Prompt for shares", '6', func() {
						_, _, p1Weight, p2Weight, totalWeight := itemParticipantAmounts(*itemPtr)
						p1SharesInit := ""
						p2SharesInit := ""
						if totalWeight > 0 {
							p1Reduced, p2Reduced := reduceRatio(p1Weight, p2Weight)
							p1SharesInit = strconv.Itoa(p1Reduced)
							p2SharesInit = strconv.Itoa(p2Reduced)
						}
						showPromptForm("Shares Split", []string{"Shares owed by " + p1, "Shares owed by " + p2}, []string{p1SharesInit, p2SharesInit}, func(vals []string) bool {
							p1Raw := strings.TrimSpace(vals[0])
							p2Raw := strings.TrimSpace(vals[1])
							if p1Raw == "" && p2Raw == "" {
								setFocus(list)
								return false
							}

							p1Shares := 0
							p2Shares := 0
							if p1Raw != "" {
								parsed, err := strconv.Atoi(p1Raw)
								if err != nil || parsed < 0 {
									setFocus(list)
									return false
								}
								p1Shares = parsed
							}
							if p2Raw != "" {
								parsed, err := strconv.Atoi(p2Raw)
								if err != nil || parsed < 0 {
									setFocus(list)
									return false
								}
								p2Shares = parsed
							}

							switch {
							case p1Raw == "":
								p1Shares = 1
							case p2Raw == "":
								p2Shares = 1
							}

							if p1Shares <= 0 || p2Shares <= 0 {
								setFocus(list)
								return false
							}

							itemPtr.SharedWith = buildWeightedSharedWith(p1Shares, p2Shares)
							closeModal()
							refreshItemsTable()
							return true
						}, func() { setFocus(list) })
					})
					list.AddItem("Exact amounts owed", "Prompt for exact amounts", '7', func() {
						p1Amount, p2Amount, _, _, totalWeight := itemParticipantAmounts(*itemPtr)
						p1AmountInit := ""
						p2AmountInit := ""
						if totalWeight > 0 {
							p1AmountInit = formatMoney(p1Amount)
							p2AmountInit = formatMoney(p2Amount)
						}
						showPromptForm("Exact Amounts", []string{"Amount owed by " + p1, "Amount owed by " + p2}, []string{p1AmountInit, p2AmountInit}, func(vals []string) bool {
							p1Raw := strings.TrimSpace(vals[0])
							p2Raw := strings.TrimSpace(vals[1])
							if p1Raw == "" && p2Raw == "" {
								setFocus(list)
								return false
							}

							totalAmount, err := strconv.ParseFloat(itemPtr.Amount, 64)
							if err != nil || !isFiniteNumber(totalAmount) || totalAmount < 0 {
								setFocus(list)
								return false
							}
							totalCents := int(math.Round(totalAmount * 100))

							p1Cents := 0
							p2Cents := 0
							if p1Raw != "" {
								parsed, err := strconv.ParseFloat(p1Raw, 64)
								if err != nil || !isFiniteNumber(parsed) || parsed < 0 {
									setFocus(list)
									return false
								}
								p1Cents = int(math.Round(parsed * 100))
							}
							if p2Raw != "" {
								parsed, err := strconv.ParseFloat(p2Raw, 64)
								if err != nil || !isFiniteNumber(parsed) || parsed < 0 {
									setFocus(list)
									return false
								}
								p2Cents = int(math.Round(parsed * 100))
							}

							switch {
							case p1Raw == "":
								p1Cents = totalCents - p2Cents
							case p2Raw == "":
								p2Cents = totalCents - p1Cents
							}

							if p1Cents < 0 || p2Cents < 0 || p1Cents+p2Cents != totalCents {
								setFocus(list)
								return false
							}

							itemPtr.SharedWith = buildWeightedSharedWith(p1Cents, p2Cents)
							closeModal()
							refreshItemsTable()
							return true
						}, func() { setFocus(list) })
					})

					initQtyInt, initDesc := splitwise.ParseItemDescription(itemPtr.Description)
					if initQtyInt <= 0 {
						initQtyInt = 1
					}
					initDisplayDesc, initExtra := splitTUIDescriptionExtra(initDesc)
					initQty := strconv.Itoa(initQtyInt)
					initAmount, _ := strconv.ParseFloat(itemPtr.Amount, 64)
					initPerItemCost := formatMoney(initAmount / float64(initQtyInt))

					list.AddItem("Edit item", "Edit quantity, description, and unit cost", 'e', func() {
						showPromptForm("Edit Item", []string{"Qty", "Description", "Per Item Cost"}, []string{initQty, initDisplayDesc, initPerItemCost}, func(vals []string) bool {
							qty := 1
							if parsedQty, err := strconv.Atoi(strings.TrimSpace(vals[0])); err == nil && parsedQty > 0 {
								qty = parsedQty
							}
							perItemCost, err := strconv.ParseFloat(strings.TrimSpace(vals[2]), 64)
							if err != nil || !isFiniteNumber(perItemCost) || perItemCost < 0 {
								setFocus(list)
								return false
							}
							newDesc := strings.TrimSpace(vals[1])
							if initExtra != "" {
								newDesc += " | " + initExtra
							}
							itemPtr.Description = splitwise.FormatItemDescription(qty, newDesc)
							itemPtr.Amount = formatMoney(perItemCost * float64(qty))
							refreshItemsTable()
							closeModal()
							return true
						}, func() { setFocus(list) })
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
					list.AddItem("Split item", "Split out N quantity units into a new row", 's', func() {
						showPromptForm("Split Item", []string{"Qty to split out (e.g. 1)"}, nil, func(vals []string) bool {
							currentQty, currentDesc := splitwise.ParseItemDescription(itemPtr.Description)
							currentBaseDesc, currentExtra := splitTUIDescriptionExtra(currentDesc)
							if currentQty <= 1 {
								setFocus(list)
								return false
							}
							splitQty, err := strconv.Atoi(strings.TrimSpace(vals[0]))
							if err != nil || splitQty <= 0 || splitQty >= currentQty {
								setFocus(list)
								return false
							}

							totalAmount, _ := strconv.ParseFloat(itemPtr.Amount, 64)
							if !isFiniteNumber(totalAmount) || totalAmount < 0 {
								setFocus(list)
								return false
							}
							perItemCost := totalAmount / float64(currentQty)
							remainingQty := currentQty - splitQty
							serializedDesc := currentBaseDesc
							if currentExtra != "" {
								serializedDesc += " | " + currentExtra
							}

							newItem := splitwise.Item{
								Description: splitwise.FormatItemDescription(splitQty, serializedDesc),
								Amount:      formatMoney(perItemCost * float64(splitQty)),
								SharedWith:  append([]string{}, itemPtr.SharedWith...),
							}

							if len(itemPtr.SharedWith) == currentQty {
								newItem.SharedWith = append([]string{}, itemPtr.SharedWith[:splitQty]...)
								itemPtr.SharedWith = append([]string{}, itemPtr.SharedWith[splitQty:]...)
							}

							itemPtr.Description = splitwise.FormatItemDescription(remainingQty, serializedDesc)
							itemPtr.Amount = formatMoney(perItemCost * float64(remainingQty))
							parsedDetails.Items = append(parsedDetails.Items[:idx+1], append([]splitwise.Item{newItem}, parsedDetails.Items[idx+1:]...)...)
							closeModal()
							refreshItemsTable()
							return true
						}, func() { setFocus(list) })
					})
					list.AddItem("No one owes this item", "", '0', func() {
						itemPtr.SharedWith = nil
						refreshItemsTable()
						closeModal()
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
					setFocus(list)
				}
			}(i)
			row++
		}

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("---").SetSelectable(false).SetTextColor(tcell.ColorGreen))
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("---").SetSelectable(false).SetTextColor(tcell.ColorGreen))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell("---").SetSelectable(false).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Subtotal").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell(fmt.Sprintf("%.2f", subtotalRender)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
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
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell(fmt.Sprintf("%.2f", taxTotalRender)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Tip").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell(fmt.Sprintf("%.2f", tipTotalRender)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
		row++

		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("Total").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell(formattedCalculatedTotal).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
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
			if showPerItemColumn {
				itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
			}
			itemsTable.SetCell(row, totalCol, tview.NewTableCell(fmt.Sprintf("%s != %s", formattedFreshTotalPaid, formattedCalculatedTotal)).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorRed))
			itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
			row++
		}

		// Add "Amounts Paid" directly into the items table
		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("--- Amounts Paid ---").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
		row++

		for i, eu := range expense.Users {
			lastName := ""
			if eu.User.LastName != nil {
				lastName = *eu.User.LastName
			}
			name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))

			itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
			itemsTable.SetCell(row, 1, tview.NewTableCell(name).SetSelectable(true).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			if showPerItemColumn {
				itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(true))
			}
			itemsTable.SetCell(row, totalCol, tview.NewTableCell(eu.PaidShare).SetSelectable(true).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorWhite))
			itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(true))

			rowActions[row] = func(idx int, userName string) func() {
				return func() {
					focusBeforeModal = itemsTable
					isModalOpen = true
					euPtr := &expense.Users[idx]

					closeModal := func() {
						pages.RemovePage("split_modal")
						isModalOpen = false
						setFocus(focusBeforeModal)
					}

					list := tview.NewList().ShowSecondaryText(true)
					list.SetBorder(true).SetTitle(fmt.Sprintf("Actions: Paid amount for %s", userName))

					list.AddItem("Edit amount", "Manually input paid amount", '1', func() {
						showPromptForm(fmt.Sprintf("Edit Paid Amount: %s", userName), []string{"Paid Amount"}, []string{euPtr.PaidShare}, func(vals []string) bool {
							euPtr.PaidShare = vals[0]
							refreshItemsTable()
							closeModal()
							return true
						}, func() {
							setFocus(list)
						})
					})

					list.AddItem("Paid %", "Calculate paid based on total cost", '2', func() {
						showPromptForm(fmt.Sprintf("Paid %%: %s", userName), []string{"Percentage (%)"}, []string{"100"}, func(vals []string) bool {
							perc, err := strconv.ParseFloat(vals[0], 64)
							if err == nil {
								euPtr.PaidShare = fmt.Sprintf("%.2f", calculatedTotal*(perc/100.0))
								refreshItemsTable()
							}
							closeModal()
							return true
						}, func() {
							setFocus(list)
						})
					})

					list.AddItem("Paid full amount", "Set to calculated total", '3', func() {
						// Zero out everyone else
						for ui := range expense.Users {
							if ui != idx {
								expense.Users[ui].PaidShare = "0.00"
							}
						}
						euPtr.PaidShare = fmt.Sprintf("%.2f", calculatedTotal)
						expense.Cost = euPtr.PaidShare
						form.GetFormItemByLabel("Cost").(*tview.InputField).SetText(expense.Cost)
						refreshItemsTable()
						closeModal()
					})

					if math.Abs(calculatedTotal-freshTotalPaid) > 0.009 {
						list.AddItem("Paid remainder", "Adjust by the remaining delta to match total", '4', func() {
							currentPaid, _ := strconv.ParseFloat(euPtr.PaidShare, 64)
							euPtr.PaidShare = fmt.Sprintf("%.2f", currentPaid+(calculatedTotal-freshTotalPaid))
							refreshItemsTable()
							closeModal()
						})
					}

					list.AddItem("Paid nothing", "Set paid amount to 0", '5', func() {
						euPtr.PaidShare = "0.00"
						refreshItemsTable()
						closeModal()
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
							AddItem(list, 15, 1, true).
							AddItem(nil, 0, 1, false), 60, 1, true).
						AddItem(nil, 0, 1, false)

					pages.AddPage("split_modal", modal, true, true)
					setFocus(list)
				}
			}(i, name)
			row++
		}

		// Add "Amounts Owed" Breakdown
		itemsTable.SetCell(row, 0, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, 1, tview.NewTableCell("--- Amounts Owed ---").SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
		if showPerItemColumn {
			itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
		}
		itemsTable.SetCell(row, totalCol, tview.NewTableCell("").SetSelectable(false))
		itemsTable.SetCell(row, sharedCol, tview.NewTableCell("").SetSelectable(false))
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
			if showPerItemColumn {
				itemsTable.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
			}
			itemsTable.SetCell(row, totalCol, tview.NewTableCell(eu.OwedShare).SetSelectable(false).SetAlign(tview.AlignRight).SetTextColor(tcell.ColorGreen))
			itemsTable.SetCell(row, sharedCol, tview.NewTableCell(balanceText).SetSelectable(false).SetAlign(tview.AlignLeft).SetTextColor(tcell.ColorGreen))
			row++
		}
	}

	refreshItemsTable()

	itemsTable.SetSelectedFunc(func(row, _ int) {
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
					setFocus(focusables[next])
					return nil
				}
			}
			setFocus(focusables[0])
			return nil
		} else if event.Key() == tcell.KeyBacktab {
			currentFocus := app.GetFocus()
			for i, p := range focusables {
				if p == currentFocus {
					next := (i - 1 + len(focusables)) % len(focusables)
					setFocus(focusables[next])
					return nil
				}
			}
			setFocus(focusables[len(focusables)-1])
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
		return false, nil, err
	}

	return sent, sendResponse, nil
}
