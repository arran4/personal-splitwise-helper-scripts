package splitwise

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ExpensesResponse struct {
	Expenses []Expense `json:"expenses"`
}

type ExpenseResponse struct {
	Expense DetailedExpense `json:"expense"`
}

type Expense struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Cost        string `json:"cost"`
	Currency    string `json:"currency_code"`
	Date        string `json:"date"`
}

type DetailedExpense struct {
	ID                     int           `json:"id"`
	GroupID                interface{}   `json:"group_id"`
	ExpenseBundleID        interface{}   `json:"expense_bundle_id"`
	Description            string        `json:"description"`
	Repeats                bool          `json:"repeats"`
	RepeatInterval         interface{}   `json:"repeat_interval"`
	EmailReminder          bool          `json:"email_reminder"`
	EmailReminderInAdvance int           `json:"email_reminder_in_advance"`
	NextRepeat             interface{}   `json:"next_repeat"`
	Details                string        `json:"details"`
	CommentsCount          int           `json:"comments_count"`
	Payment                bool          `json:"payment"`
	CreationMethod         string        `json:"creation_method"`
	TransactionMethod      string        `json:"transaction_method"`
	TransactionConfirmed   bool          `json:"transaction_confirmed"`
	TransactionID          interface{}   `json:"transaction_id"`
	TransactionStatus      interface{}   `json:"transaction_status"`
	Cost                   string        `json:"cost"`
	CurrencyCode           string        `json:"currency_code"`
	Repayments             []Repayment   `json:"repayments"`
	Date                   string        `json:"date"`
	CreatedAt              string        `json:"created_at"`
	CreatedBy              User          `json:"created_by"`
	UpdatedAt              string        `json:"updated_at"`
	UpdatedBy              *User         `json:"updated_by"`
	DeletedAt              interface{}   `json:"deleted_at"`
	DeletedBy              *User         `json:"deleted_by"`
	Category               Category      `json:"category"`
	Receipt                Receipt       `json:"receipt"`
	Users                  []ExpenseUser `json:"users"`
}

type Repayment struct {
	From   int    `json:"from"`
	To     int    `json:"to"`
	Amount string `json:"amount"`
}

type Receipt struct {
	Large    interface{} `json:"large"`
	Original interface{} `json:"original"`
}

type Picture struct {
	Medium string `json:"medium"`
}

type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ExpenseUser struct {
	User       User   `json:"user"`
	UserID     int    `json:"user_id"`
	PaidShare  string `json:"paid_share"`
	OwedShare  string `json:"owed_share"`
	NetBalance string `json:"net_balance"`
}

type User struct {
	ID        int     `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  *string `json:"last_name"`
	Picture   Picture `json:"picture"`
}

type ItemizedDetail struct {
	Notes string
	Items []Item
	Tax   []PersonAmount
	Tip   []PersonAmount
}

type Item struct {
	Description string
	Amount      string
	SharedWith  []string
}

type PersonAmount struct {
	Name   string
	Amount string
}

func SerializeDetails(details *ItemizedDetail) string {
	if details == nil {
		return ""
	}

	var lines []string
	for _, item := range details.Items {
		lines = append(lines, serializeItemLines(item)...)
	}
	if len(details.Tax) > 0 {
		lines = append(lines, "Tax: "+serializePersonAmounts(details.Tax))
	}
	if len(details.Tip) > 0 {
		lines = append(lines, "Tip: "+serializePersonAmounts(details.Tip))
	}

	notes := strings.TrimSpace(details.Notes)
	itemsBlock := strings.Join(lines, "\n")
	switch {
	case notes != "" && itemsBlock != "":
		return notes + "\n\n" + itemsBlock
	case notes != "":
		return notes
	default:
		return itemsBlock
	}
}

func ParseItemDescription(description string) (int, string) {
	qty := 1
	desc := strings.TrimSpace(description)
	if parts := strings.SplitN(desc, "x ", 2); len(parts) == 2 {
		if parsedQty, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && parsedQty > 0 {
			qty = parsedQty
			desc = strings.TrimSpace(parts[1])
		}
	}
	return qty, desc
}

func FormatItemDescription(qty int, description string) string {
	description = strings.TrimSpace(description)
	if qty <= 1 {
		return description
	}
	return fmt.Sprintf("%dx %s", qty, description)
}

func ParseDetails(details string) *ItemizedDetail {
	if details == "" {
		return nil
	}

	result := &ItemizedDetail{}

	// Split by double newline to separate notes from items
	parts := strings.Split(details, "\n\n")

	var itemsStr string

	// Check if the last part looks like items
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if isItemsFormat(lastPart) {
			itemsStr = lastPart
			if len(parts) > 1 {
				result.Notes = strings.Join(parts[:len(parts)-1], "\n\n")
			}
		} else {
			result.Notes = details
			return result
		}
	}

	if itemsStr != "" {
		lines := strings.Split(itemsStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "Tax: ") {
				result.Tax = parsePersonAmounts(strings.TrimPrefix(line, "Tax: "))
			} else if strings.HasPrefix(line, "Tip: ") {
				result.Tip = parsePersonAmounts(strings.TrimPrefix(line, "Tip: "))
			} else {
				parts := strings.SplitN(line, " - ", 2)
				if len(parts) != 2 {
					continue
				}
				desc := parts[0]

				rest := parts[1]
				amountAndPeople := strings.SplitN(rest, " (", 2)
				if len(amountAndPeople) != 2 {
					continue
				}
				amount := amountAndPeople[0]
				peopleStr := strings.TrimSuffix(amountAndPeople[1], ")")
				people := parseSharedWith(peopleStr, strings.TrimSpace(amount))

				result.Items = append(result.Items, Item{
					Description: strings.TrimSpace(desc),
					Amount:      strings.TrimSpace(amount),
					SharedWith:  people,
				})
			}
		}
	}

	return result
}

func isItemsFormat(s string) bool {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 {
		return false
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Tax: ") || strings.HasPrefix(line, "Tip: ") {
			return true // Tax/Tip is a strong signal
		}
		// Basic check for "Desc - Amount (Person...)"
		if strings.Contains(line, " - ") && strings.Contains(line, " (") && strings.HasSuffix(line, ")") {
			return true // At least one valid item line is enough
		}
	}
	return false
}

func parsePersonAmounts(s string) []PersonAmount {
	var amounts []PersonAmount
	parts := strings.Split(s, ", ")
	for _, part := range parts {
		subParts := strings.SplitN(part, " - ", 2)
		if len(subParts) == 2 {
			amounts = append(amounts, PersonAmount{
				Name:   strings.TrimSpace(subParts[0]),
				Amount: strings.TrimSpace(subParts[1]),
			})
		}
	}
	return amounts
}

func serializePersonAmounts(amounts []PersonAmount) string {
	var parts []string
	for _, amount := range amounts {
		parts = append(parts, fmt.Sprintf("%s - %s", strings.TrimSpace(amount.Name), strings.TrimSpace(amount.Amount)))
	}
	return strings.Join(parts, ", ")
}

func parseSharedWith(s, amount string) []string {
	totalAmount, _ := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	totalCents := int(math.Round(totalAmount * 100))
	if totalCents > 0 {
		if weighted := parseWeightedAmountsSharedWith(s, totalCents); len(weighted) > 0 {
			return weighted
		}
	}

	var people []string
	for _, rawPart := range strings.Split(s, ",") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}

		name := part
		count := 1
		if idx := strings.LastIndex(part, "*"); idx > 0 {
			if parsedCount, err := strconv.Atoi(strings.TrimSpace(part[idx+1:])); err == nil && parsedCount > 0 {
				name = strings.TrimSpace(part[:idx])
				count = parsedCount
			}
		}
		for i := 0; i < count; i++ {
			people = append(people, name)
		}
	}
	return people
}

func parseWeightedAmountsSharedWith(s string, totalCents int) []string {
	var order []string
	amounts := make(map[string]int)
	consumed := 0
	for _, rawPart := range strings.Split(s, ",") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		idx := strings.LastIndex(part, " $")
		if idx <= 0 {
			return nil
		}
		name := strings.TrimSpace(part[:idx])
		value := strings.TrimSpace(part[idx+2:])
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed < 0 {
			return nil
		}
		cents := int(math.Round(parsed * 100))
		if _, ok := amounts[name]; !ok {
			order = append(order, name)
		}
		amounts[name] += cents
		consumed += cents
	}
	if len(order) == 0 {
		return nil
	}
	if consumed != totalCents {
		diff := totalCents - consumed
		amounts[order[len(order)-1]] += diff
	}

	var people []string
	for _, name := range order {
		for i := 0; i < amounts[name]; i++ {
			people = append(people, name)
		}
	}
	return people
}

func serializeSharedWith(sharedWith []string, totalAmount float64, qty int) string {
	counts := make(map[string]int)
	var ordered []string
	for _, person := range sharedWith {
		if counts[person] == 0 {
			ordered = append(ordered, person)
		}
		counts[person]++
	}

	var parts []string
	if len(sharedWith) == qty {
		for _, person := range ordered {
			if counts[person] > 1 {
				parts = append(parts, fmt.Sprintf("%s*%d", person, counts[person]))
				continue
			}
			parts = append(parts, person)
		}
		return strings.Join(parts, ", ")
	}

	if len(sharedWith) == 0 {
		return ""
	}
	shareValue := totalAmount / float64(len(sharedWith))
	for _, person := range ordered {
		parts = append(parts, fmt.Sprintf("%s $%.2f", person, shareValue*float64(counts[person])))
	}
	return strings.Join(parts, ", ")
}

func serializeItemLines(item Item) []string {
	description := strings.TrimSpace(item.Description)
	amount := strings.TrimSpace(item.Amount)
	totalAmount, _ := strconv.ParseFloat(amount, 64)
	qty, baseDesc := ParseItemDescription(description)

	if len(item.SharedWith) == 0 {
		return []string{fmt.Sprintf("%s - %s ()", description, amount)}
	}

	counts := make(map[string]int)
	var ordered []string
	for _, person := range item.SharedWith {
		if counts[person] == 0 {
			ordered = append(ordered, person)
		}
		counts[person]++
	}

	if len(ordered) == len(item.SharedWith) {
		return []string{fmt.Sprintf("%s - %s (%s)", description, amount, strings.Join(ordered, ", "))}
	}

	if len(item.SharedWith) == qty {
		var lines []string
		for _, person := range ordered {
			personQty := counts[person]
			personDesc := FormatItemDescription(personQty, baseDesc)
			personAmount := totalAmount * float64(personQty) / float64(qty)
			lines = append(lines, fmt.Sprintf("%s - %.2f (%s)", personDesc, personAmount, person))
		}
		return lines
	}

	weightTotal := len(item.SharedWith)
	var lines []string
	for _, person := range ordered {
		personAmount := totalAmount * float64(counts[person]) / float64(weightTotal)
		lines = append(lines, fmt.Sprintf("%s - %.2f (%s)", baseDesc, personAmount, person))
	}
	return lines
}

// CalculateOwed amounts based on ItemizedDetail and list of Users.
// Updates expense.Users in-place.
func CalculateOwed(expense *DetailedExpense, details *ItemizedDetail) {
	if details == nil || expense == nil {
		return
	}

	userMap := make(map[string]*ExpenseUser)
	for i, eu := range expense.Users {
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))
		userMap[name] = &expense.Users[i]

		// Reset owed to 0 for recalculation
		expense.Users[i].OwedShare = "0.00"
	}

	// Add items
	for _, item := range details.Items {
		cost, err := strconv.ParseFloat(item.Amount, 64)
		if err != nil || len(item.SharedWith) == 0 {
			continue
		}
		splitAmt := cost / float64(len(item.SharedWith))
		for _, personName := range item.SharedWith {
			if user, ok := userMap[personName]; ok {
				currentOwed, _ := strconv.ParseFloat(user.OwedShare, 64)
				user.OwedShare = fmt.Sprintf("%.2f", currentOwed+splitAmt)
			}
		}
	}

	// Add tax
	for _, item := range details.Tax {
		if user, ok := userMap[item.Name]; ok {
			cost, _ := strconv.ParseFloat(item.Amount, 64)
			currentOwed, _ := strconv.ParseFloat(user.OwedShare, 64)
			user.OwedShare = fmt.Sprintf("%.2f", currentOwed+cost)
		}
	}

	// Add tip
	for _, item := range details.Tip {
		if user, ok := userMap[item.Name]; ok {
			cost, _ := strconv.ParseFloat(item.Amount, 64)
			currentOwed, _ := strconv.ParseFloat(user.OwedShare, 64)
			user.OwedShare = fmt.Sprintf("%.2f", currentOwed+cost)
		}
	}

	// Update NetBalance
	for i := range expense.Users {
		paid, _ := strconv.ParseFloat(expense.Users[i].PaidShare, 64)
		owed, _ := strconv.ParseFloat(expense.Users[i].OwedShare, 64)
		expense.Users[i].NetBalance = fmt.Sprintf("%.2f", paid-owed)
	}
}

// AutoCorrectPaidShares updates the PaidShare amounts when the calculated total changes.
// If no one has paid anything yet, the full amount is assigned to the first user.
// If exactly one person has paid, their amount is adjusted by the delta between totals.
func AutoCorrectPaidShares(expense *DetailedExpense, previousTotal, calculatedTotal float64) {
	if expense == nil || len(expense.Users) == 0 {
		return
	}

	formattedCalculatedTotal := fmt.Sprintf("%.2f", calculatedTotal)
	formattedPreviousTotal := fmt.Sprintf("%.2f", previousTotal)

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

	if formattedPreviousTotal != formattedCalculatedTotal && formattedTotalPaid != formattedCalculatedTotal {
		if math.Abs(totalPaid) < 0.01 {
			// No one is paying, assign all to first payee
			expense.Users[0].PaidShare = formattedCalculatedTotal
			expense.Cost = formattedCalculatedTotal
		} else if paidCounts == 1 {
			// Only 1 person has paid, preserve their manual offset by applying the total delta.
			paid, _ := strconv.ParseFloat(expense.Users[onlyPaidIdx].PaidShare, 64)
			expense.Users[onlyPaidIdx].PaidShare = fmt.Sprintf("%.2f", paid+(calculatedTotal-previousTotal))
			expense.Cost = formattedCalculatedTotal
		}
		// If multiple people paid, leave it alone (UI will warn)
	}
}
