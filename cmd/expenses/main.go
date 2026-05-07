package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
)

const CacheDir = ".cache"

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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: expenses <list|show> [options]")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "list":
		listCmd := flag.NewFlagSet("list", flag.ExitOnError)

		groupID := listCmd.Int("group-id", 0, "Only expenses in that group will be returned")
		friendID := listCmd.Int("friend-id", 0, "Only expenses between current and provided user")
		datedAfter := listCmd.String("dated-after", "", "Dated after")
		datedBefore := listCmd.String("dated-before", "", "Dated before")
		updatedAfter := listCmd.String("updated-after", "", "Updated after")
		updatedBefore := listCmd.String("updated-before", "", "Updated before")
		limit := listCmd.Int("limit", 20, "Number of expenses to fetch per page")
		offset := listCmd.Int("offset", 0, "Initial offset")
		keepGoing := listCmd.Bool("keep-going", false, "Fetch all pages")
		getPages := listCmd.Int("get-pages", 1, "Number of pages to fetch (ignored if --keep-going is set)")

		listCmd.Parse(os.Args[2:])

		client, err := splitwise.NewClient()
		if err != nil {
			fmt.Println("Error initializing client:", err)
			os.Exit(1)
		}

		var allExpenses []Expense
		currentOffset := *offset
		pagesFetched := 0

		for {
			params := url.Values{}
			if *groupID != 0 {
				params.Add("group_id", strconv.Itoa(*groupID))
			}
			if *friendID != 0 {
				params.Add("friend_id", strconv.Itoa(*friendID))
			}
			if *datedAfter != "" {
				params.Add("dated_after", *datedAfter)
			}
			if *datedBefore != "" {
				params.Add("dated_before", *datedBefore)
			}
			if *updatedAfter != "" {
				params.Add("updated_after", *updatedAfter)
			}
			if *updatedBefore != "" {
				params.Add("updated_before", *updatedBefore)
			}

			params.Add("limit", strconv.Itoa(*limit))
			params.Add("offset", strconv.Itoa(currentOffset))

			data, err := client.GetExpenses(params.Encode())
			if err != nil {
				fmt.Println("Error fetching expenses:", err)
				os.Exit(1)
			}

			var resp ExpensesResponse
			if err := json.Unmarshal(data, &resp); err != nil {
				fmt.Println("Error parsing expenses:", err)
				os.Exit(1)
			}

			allExpenses = append(allExpenses, resp.Expenses...)
			pagesFetched++

			if len(resp.Expenses) < *limit {
				break // No more data available
			}

			if !*keepGoing && pagesFetched >= *getPages {
				break
			}

			currentOffset += *limit
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tDATE\tDESCRIPTION\tCOST")
		for _, e := range allExpenses {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s %s\n", e.ID, e.Date, e.Description, e.Cost, e.Currency)
		}
		w.Flush()

	case "show":
		showCmd := flag.NewFlagSet("show", flag.ExitOnError)
		id := showCmd.String("id", "", "ID of the expense to show")
		refresh := showCmd.Bool("refresh", false, "Force refresh from API instead of using cache")

		showCmd.Parse(os.Args[2:])

		if *id == "" {
			fmt.Println("Please provide an expense ID via --id")
			os.Exit(1)
		}

		if err := os.MkdirAll(CacheDir, 0755); err != nil {
			fmt.Println("Error creating cache directory:", err)
			os.Exit(1)
		}

		cacheFile := filepath.Join(CacheDir, fmt.Sprintf("expense_%s.json", *id))
		var data []byte
		var err error

		if !*refresh {
			data, err = os.ReadFile(cacheFile)
		}

		if *refresh || os.IsNotExist(err) || len(data) == 0 {
			client, err := splitwise.NewClient()
			if err != nil {
				fmt.Println("Error initializing client:", err)
				os.Exit(1)
			}

			data, err = client.GetExpense(*id)
			if err != nil {
				fmt.Println("Error fetching expense:", err)
				os.Exit(1)
			}

			if err := os.WriteFile(cacheFile, data, 0644); err != nil {
				fmt.Println("Warning: could not write to cache file:", err)
			}
		} else if err != nil {
			fmt.Println("Error reading from cache:", err)
			os.Exit(1)
		}

		var resp ExpenseResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Println("Error parsing expense JSON:", err)
			os.Exit(1)
		}

		e := resp.Expense

		fmt.Printf("=========================================\n")
		fmt.Printf(" EXPENSE DETAILS (ID: %d)\n", e.ID)
		fmt.Printf("=========================================\n")
		fmt.Printf(" Description : %s\n", e.Description)
		fmt.Printf(" Total Cost  : %s %s\n", e.Cost, e.CurrencyCode)
		fmt.Printf(" Category    : %s\n", e.Category.Name)
		fmt.Printf(" Date        : %s\n", e.Date)
		fmt.Printf(" Created     : %s\n", e.CreatedAt)
		if e.Details != "" {
			fmt.Printf(" Notes       : %s\n", e.Details)
		}

		if e.CreationMethod == "itemized" && e.Details != "" {
			itemizedDetails := parseDetails(e.Details)
			if itemizedDetails != nil {
				fmt.Printf("-----------------------------------------\n")
				fmt.Printf(" ITEMIZED BREAKDOWN\n")
				fmt.Printf("-----------------------------------------\n")
				itemW := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				if len(itemizedDetails.Items) > 0 {
					fmt.Fprintln(itemW, "ITEM\tCOST\tSHARED WITH")
					for _, item := range itemizedDetails.Items {
						fmt.Fprintf(itemW, "%s\t%s\t%s\n", item.Description, item.Amount, strings.Join(item.SharedWith, ", "))
					}
				}
				if len(itemizedDetails.Tax) > 0 {
					fmt.Fprintln(itemW, "\nTAX")
					for _, pa := range itemizedDetails.Tax {
						fmt.Fprintf(itemW, "%s\t%s\n", pa.Name, pa.Amount)
					}
				}
				if len(itemizedDetails.Tip) > 0 {
					fmt.Fprintln(itemW, "\nTIP")
					for _, pa := range itemizedDetails.Tip {
						fmt.Fprintf(itemW, "%s\t%s\n", pa.Name, pa.Amount)
					}
				}
				itemW.Flush()
			}
		}

		fmt.Printf("-----------------------------------------\n")
		fmt.Printf(" USER BREAKDOWN\n")
		fmt.Printf("-----------------------------------------\n")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "USER\tPAID\tOWED\tNET")
		for _, eu := range e.Users {
			lastName := ""
			if eu.User.LastName != nil {
				lastName = *eu.User.LastName
			}
			name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, eu.PaidShare, eu.OwedShare, eu.NetBalance)
		}
		w.Flush()
		fmt.Printf("=========================================\n")

	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}

func parseDetails(details string) *ItemizedDetail {
	if details == "" {
		return nil
	}

	result := &ItemizedDetail{}
	lines := strings.Split(details, "\n")

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

	return result
}

func parsePersonAmounts(s string) []PersonAmount {
	var amounts []PersonAmount
	parts := strings.Split(s, ", ")
	for _, part := range parts {
		subParts := strings.SplitN(part, " - ", 2)
		if len(subParts) == 2 {
			amounts = append(amounts, PersonAmount{
				Name:   subParts[0],
				Amount: subParts[1],
			})
		}
	}
	return amounts
}
