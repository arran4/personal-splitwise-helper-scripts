package splitwise

import (
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
				people := strings.Split(peopleStr, ", ")

				result.Items = append(result.Items, Item{
					Description: desc,
					Amount:      amount,
					SharedWith:  people,
				})
			}
		}
	}

	return result
}

func isItemsFormat(s string) bool {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Tax: ") || strings.HasPrefix(line, "Tip: ") {
			continue
		}
		// Basic check for "Desc - Amount (Person...)"
		if !strings.Contains(line, " - ") || !strings.Contains(line, " (") || !strings.HasSuffix(line, ")") {
			return false
		}
	}
	return true
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
