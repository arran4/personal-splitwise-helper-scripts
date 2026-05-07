package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
)

type ExpensesResponse struct {
	Expenses []Expense `json:"expenses"`
}

type Expense struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Cost        string `json:"cost"`
	Currency    string `json:"currency_code"`
	Date        string `json:"date"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: expenses <list> [options]")
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

	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}
