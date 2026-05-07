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
	"github.com/arran4/personal-splitwise-helper-scripts/pkg/tui"
)

const CacheDir = ".cache"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: expenses <list|show|edit> [options]")
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

		var allExpenses []splitwise.Expense
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

			var resp splitwise.ExpensesResponse
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

		resp, err := fetchExpense(*id, *refresh)
		if err != nil {
			fmt.Println(err)
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
			itemizedDetails := splitwise.ParseDetails(e.Details)
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

	case "edit":
		editCmd := flag.NewFlagSet("edit", flag.ExitOnError)
		id := editCmd.String("id", "", "ID of the expense to edit")
		refresh := editCmd.Bool("refresh", false, "Force refresh from API instead of using cache")

		editCmd.Parse(os.Args[2:])

		if *id == "" {
			fmt.Println("Please provide an expense ID via --id")
			os.Exit(1)
		}

		resp, err := fetchExpense(*id, *refresh)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Lazily fetch current user if not cached
		if _, err := os.Stat(filepath.Join(CacheDir, "current_user.json")); os.IsNotExist(err) {
			fmt.Println("Current user not cached, fetching...")
			if err := fetchCurrentUser(); err != nil {
				fmt.Println("Could not fetch current user:", err)
				os.Exit(1)
			}
		}

		sent, err := tui.EditExpense(&resp.Expense)
		if err != nil {
			fmt.Println("Error running TUI:", err)
			os.Exit(1)
		}
		if sent {
			if err := invalidateExpenseCache(*id); err != nil {
				fmt.Println("Warning: could not invalidate cache:", err)
			}
			fmt.Println("success")
		}

	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}

func invalidateExpenseCache(id string) error {
	cacheFile := filepath.Join(CacheDir, fmt.Sprintf("expense_%s.json", id))
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func fetchExpense(id string, refresh bool) (*splitwise.ExpenseResponse, error) {
	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating cache directory: %w", err)
	}

	cacheFile := filepath.Join(CacheDir, fmt.Sprintf("expense_%s.json", id))
	var data []byte
	var err error

	if !refresh {
		data, err = os.ReadFile(cacheFile)
	}

	if refresh || os.IsNotExist(err) || len(data) == 0 {
		client, err := splitwise.NewClient()
		if err != nil {
			return nil, fmt.Errorf("error initializing client: %w", err)
		}

		data, err = client.GetExpense(id)
		if err != nil {
			return nil, fmt.Errorf("error fetching expense: %w", err)
		}

		if err := os.WriteFile(cacheFile, data, 0644); err != nil {
			fmt.Println("Warning: could not write to cache file:", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error reading from cache: %w", err)
	}

	var resp splitwise.ExpenseResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("error parsing expense JSON: %w", err)
	}

	return &resp, nil
}

func fetchCurrentUser() error {
	client, err := splitwise.NewClient()
	if err != nil {
		return fmt.Errorf("initializing client: %w", err)
	}

	data, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("fetching current user: %w", err)
	}

	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	cachePath := filepath.Join(CacheDir, "current_user.json")
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing to cache file: %w", err)
	}

	fmt.Printf("Successfully fetched current user and cached to %s\n", cachePath)
	return nil
}
