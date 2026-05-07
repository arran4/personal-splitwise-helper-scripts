package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SelectionOption struct {
	Label string
}

type SelectionRefreshFunc func() ([]SelectionOption, string, error)

type scoredSelectionOption struct {
	index int
	label string
	score int
}

func scoreSelectionMatch(candidate, query string) (int, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, true
	}

	queryRunes := []rune(query)
	candidateRunes := []rune(candidate)

	score := 0
	searchStart := 0
	lastMatch := -1
	for i, qr := range queryRunes {
		match := -1
		for j := searchStart; j < len(candidateRunes); j++ {
			if strings.EqualFold(string(candidateRunes[j]), string(qr)) {
				match = j
				break
			}
		}
		if match < 0 {
			return 0, false
		}
		if i == 0 {
			score += match
		} else {
			score += match - lastMatch - 1
		}
		if isUppercaseRune(qr) && !isUppercaseRune(candidateRunes[match]) {
			score++
		}
		lastMatch = match
		searchStart = match + 1
	}

	return score, true
}

func filterSelectionOptions(options []SelectionOption, query string) []scoredSelectionOption {
	filtered := make([]scoredSelectionOption, 0, len(options))
	for i, option := range options {
		score, ok := scoreSelectionMatch(option.Label, query)
		if !ok {
			continue
		}
		filtered = append(filtered, scoredSelectionOption{
			index: i,
			label: option.Label,
			score: score,
		})
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].score != filtered[j].score {
			return filtered[i].score < filtered[j].score
		}
		if len(filtered[i].label) != len(filtered[j].label) {
			return len(filtered[i].label) < len(filtered[j].label)
		}
		return strings.ToLower(filtered[i].label) < strings.ToLower(filtered[j].label)
	})

	return filtered
}

func isUppercaseRune(r rune) bool {
	return strings.ToUpper(string(r)) == string(r) && strings.ToLower(string(r)) != string(r)
}

func SelectOption(title string, options []SelectionOption, footer string, onRefresh SelectionRefreshFunc) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options available")
	}

	app := tview.NewApplication()
	selected := -1
	filterInput := tview.NewInputField().SetLabel("Filter: ")
	filterInput.SetFieldWidth(0)
	filterInput.SetBorder(true)

	table := tview.NewTable().SetSelectable(true, true)
	table.SetBorder(true).SetTitle(title)
	table.SetFixed(1, 0)
	statusView := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	statusView.SetBorder(true).SetTitle("Status")
	updateStatus := func(extra string) {
		text := footer
		if onRefresh != nil {
			if text != "" {
				text += "\n"
			}
			text += "[r] Refresh caches  [Tab]/[Backtab] Move focus"
		}
		if extra != "" {
			if text != "" {
				text += "\n"
			}
			text += extra
		}
		statusView.SetText(text)
	}
	updateStatus("")

	var current []scoredSelectionOption
	cellIndex := map[[2]int]int{}
	selectedRow, selectedCol := 1, 0
	var refreshTable func(query string)
	screenWidth := 80
	computeResultColumns := func(filtered []scoredSelectionOption) int {
		if len(filtered) == 0 {
			return 1
		}
		maxLabelWidth := len("Results")
		for _, option := range filtered {
			if len(option.label) > maxLabelWidth {
				maxLabelWidth = len(option.label)
			}
		}
		// Allow some padding and borders per column.
		columnWidth := maxLabelWidth + 4
		if columnWidth <= 0 {
			return 1
		}
		usableWidth := screenWidth - 4
		if usableWidth <= columnWidth {
			return 1
		}
		cols := usableWidth / columnWidth
		if cols < 1 {
			cols = 1
		}
		if cols > len(filtered) {
			cols = len(filtered)
		}
		return cols
	}

	refreshTable = func(query string) {
		current = filterSelectionOptions(options, query)
		resultColumns := computeResultColumns(current)
		cellIndex = map[[2]int]int{}
		table.Clear()
		table.SetCell(0, 0, tview.NewTableCell("Results").SetSelectable(false).SetTextColor(tcell.ColorGreen))
		for col := 1; col < resultColumns; col++ {
			table.SetCell(0, col, tview.NewTableCell("").SetSelectable(false))
		}
		if len(current) == 0 {
			table.SetCell(1, 0, tview.NewTableCell("No matches").SetSelectable(false).SetTextColor(tcell.ColorRed))
			for col := 1; col < resultColumns; col++ {
				table.SetCell(1, col, tview.NewTableCell("").SetSelectable(false))
			}
			selectedRow, selectedCol = 1, 0
			table.Select(selectedRow, selectedCol)
			return
		}

		for i, option := range current {
			row := (i / resultColumns) + 1
			col := i % resultColumns
			cellIndex[[2]int{row, col}] = option.index
			table.SetCell(row, col, tview.NewTableCell(option.label).SetSelectable(true))
		}

		maxRow := ((len(current) - 1) / resultColumns) + 1
		for row := 1; row <= maxRow; row++ {
			for col := 0; col < resultColumns; col++ {
				if _, ok := cellIndex[[2]int{row, col}]; !ok {
					table.SetCell(row, col, tview.NewTableCell("").SetSelectable(false))
				}
			}
		}

		if _, ok := cellIndex[[2]int{selectedRow, selectedCol}]; !ok {
			selectedRow, selectedCol = 1, 0
		}
		table.Select(selectedRow, selectedCol)
	}

	refreshTable("")

	actionButtons := tview.NewForm()
	actionButtons.SetButtonsAlign(tview.AlignCenter)

	refreshOptions := func() {
		if onRefresh == nil {
			return
		}
		newOptions, newFooter, err := onRefresh()
		if err != nil {
			updateStatus(fmt.Sprintf("[red]Refresh failed:[white] %v", err))
			return
		}
		options = newOptions
		footer = newFooter
		updateStatus("[green]Caches refreshed")
		selectedRow, selectedCol = 1, 0
		refreshTable(filterInput.GetText())
	}

	if onRefresh != nil {
		actionButtons.AddButton("Refresh", func() {
			refreshOptions()
			app.SetFocus(table)
		})
	}
	actionButtons.AddButton("Cancel", func() {
		selected = -1
		app.Stop()
	})

	filterInput.SetChangedFunc(func(text string) {
		selectedRow, selectedCol = 1, 0
		refreshTable(text)
	})
	handoffToTable := func(event *tcell.EventKey) *tcell.EventKey {
		app.SetFocus(table)
		app.QueueEvent(tcell.NewEventKey(event.Key(), event.Rune(), event.Modifiers()))
		return nil
	}
	filterInput.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter, tcell.KeyDown:
			app.SetFocus(table)
		case tcell.KeyEsc:
			selected = -1
			app.Stop()
		}
	})
	filterInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyBacktab:
			app.SetFocus(table)
			return nil
		case tcell.KeyUp, tcell.KeyDown:
			return handoffToTable(event)
		case tcell.KeyLeft, tcell.KeyRight:
			if filterInput.GetText() == "" {
				return handoffToTable(event)
			}
			return event
		case tcell.KeyEsc:
			selected = -1
			app.Stop()
			return nil
		}
		return event
	})

	table.SetSelectionChangedFunc(func(row, column int) {
		selectedRow, selectedCol = row, column
	})
	table.SetSelectedFunc(func(row, column int) {
		if idx, ok := cellIndex[[2]int{row, column}]; ok {
			selected = idx
			app.Stop()
		}
	})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			selected = -1
			app.Stop()
			return nil
		case tcell.KeyTab, tcell.KeyBacktab, tcell.KeyUp:
			if event.Key() == tcell.KeyUp && selectedRow > 1 {
				return event
			}
			app.SetFocus(filterInput)
			return nil
		case tcell.KeyLeft, tcell.KeyRight, tcell.KeyDown, tcell.KeyEnter:
			return event
		}

		if event.Rune() != 0 && event.Modifiers()&tcell.ModCtrl == 0 && event.Modifiers()&tcell.ModAlt == 0 {
			filterInput.SetText(filterInput.GetText() + string(event.Rune()))
			app.SetFocus(filterInput)
			return nil
		}
		return event
	})

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filterInput, 3, 0, true).
		AddItem(table, 0, 1, false).
		AddItem(statusView, 4, 0, false).
		AddItem(actionButtons, 3, 0, false)

	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if onRefresh != nil && (event.Key() == tcell.KeyCtrlR || event.Rune() == 'r' || event.Rune() == 'R') {
			refreshOptions()
			return nil
		}
		return event
	})
	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screenWidth, _ = screen.Size()
		refreshTable(filterInput.GetText())
		return false
	})

	if err := app.SetRoot(root, true).SetFocus(filterInput).Run(); err != nil {
		return -1, err
	}
	if selected < 0 {
		return -1, fmt.Errorf("selection cancelled")
	}
	return selected, nil
}
