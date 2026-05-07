package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
	"github.com/arran4/personal-splitwise-helper-scripts/pkg/tui"
)

const CacheDir = ".cache"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: expenses <list|show|edit|new> [options]")
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
		verbose := editCmd.Bool("verbose", false, "Print the full server success payload after send")

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

		sent, sendResponse, err := tui.EditExpense(&resp.Expense)
		if err != nil {
			fmt.Println("Error running TUI:", err)
			os.Exit(1)
		}
		if sent {
			if err := invalidateExpenseCache(*id); err != nil {
				fmt.Println("Warning: could not invalidate cache:", err)
			}
			if *verbose && len(sendResponse) > 0 {
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, sendResponse, "", "  "); err == nil {
					fmt.Println(pretty.String())
				} else {
					fmt.Println(string(sendResponse))
				}
			} else {
				fmt.Println("success")
			}
		}

	case "new":
		newCmd := flag.NewFlagSet("new", flag.ExitOnError)
		groupID := newCmd.Int("group-id", 0, "Create the expense in this group")
		friendID := newCmd.Int("friend-id", 0, "Create the expense with this friend")
		verbose := newCmd.Bool("verbose", false, "Print the full server success payload after send")

		newCmd.Parse(os.Args[2:])

		if *groupID != 0 && *friendID != 0 {
			fmt.Println("Provide only one of --group-id or --friend-id")
			os.Exit(1)
		}

		selectedGroupID := *groupID
		selectedFriendID := *friendID
		if selectedGroupID == 0 && selectedFriendID == 0 {
			var err error
			selectedGroupID, selectedFriendID, err = chooseExpenseContext()
			if err != nil {
				fmt.Println("Error choosing expense context:", err)
				os.Exit(1)
			}
		}

		expense, err := initializeNewExpense(selectedGroupID, selectedFriendID)
		if err != nil {
			fmt.Println("Error creating new expense draft:", err)
			os.Exit(1)
		}

		sent, sendResponse, err := tui.EditExpense(expense)
		if err != nil {
			fmt.Println("Error running TUI:", err)
			os.Exit(1)
		}
		if sent {
			if *verbose && len(sendResponse) > 0 {
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, sendResponse, "", "  "); err == nil {
					fmt.Println(pretty.String())
				} else {
					fmt.Println(string(sendResponse))
				}
			} else {
				fmt.Println("success")
			}
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

type expenseChoice struct {
	label    string
	groupID  int
	friendID int
}

func chooseExpenseContext() (int, int, error) {
	if err := ensureFriendsCache(); err != nil {
		return 0, 0, err
	}
	if err := ensureGroupsCache(); err != nil {
		return 0, 0, err
	}

	choices, options, footer, err := buildExpenseContextOptions()
	if err != nil {
		return 0, 0, err
	}
	selected, err := tui.SelectOption("Select Expense Context", options, footer, func() ([]tui.SelectionOption, string, error) {
		if err := refreshFriendsCache(); err != nil {
			return nil, "", err
		}
		if err := refreshGroupsCache(); err != nil {
			return nil, "", err
		}
		var refreshErr error
		choices, options, footer, refreshErr = buildExpenseContextOptions()
		if refreshErr != nil {
			return nil, "", refreshErr
		}
		return options, footer, nil
	})
	if err != nil {
		return 0, 0, err
	}

	choice := choices[selected]
	return choice.groupID, choice.friendID, nil
}

func buildExpenseContextOptions() ([]expenseChoice, []tui.SelectionOption, string, error) {
	friends, err := splitwise.GetCachedFriends(CacheDir)
	if err != nil {
		return nil, nil, "", err
	}
	groups, err := splitwise.GetCachedGroups(CacheDir)
	if err != nil {
		return nil, nil, "", err
	}
	var choices []expenseChoice
	var options []tui.SelectionOption
	for _, f := range friends {
		name := strings.TrimSpace(strings.Join([]string{f.FirstName, f.LastName}, " "))
		if name == "" {
			name = f.Email
		}
		label := fmt.Sprintf("Friend: %s (id=%d)", name, f.ID)
		choices = append(choices, expenseChoice{label: label, friendID: f.ID})
		options = append(options, tui.SelectionOption{Label: label})
	}
	for _, g := range groups {
		label := fmt.Sprintf("Group: %s (id=%d)", g.Name, g.ID)
		choices = append(choices, expenseChoice{label: label, groupID: g.ID})
		options = append(options, tui.SelectionOption{Label: label})
	}
	if len(options) == 0 {
		return nil, nil, "", fmt.Errorf("no cached friends or groups available")
	}
	return choices, options, expenseContextFooter(), nil
}

func expenseContextFooter() string {
	return fmt.Sprintf("Friends cache: %s\nGroups cache: %s", cacheAgeString(filepath.Join(CacheDir, "friends.json")), cacheAgeString(filepath.Join(CacheDir, "groups.json")))
}

func cacheAgeString(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	age := time.Since(info.ModTime()).Round(time.Second)
	return age.String() + " old"
}

func ensureFriendsCache() error {
	if _, err := os.Stat(filepath.Join(CacheDir, "friends.json")); os.IsNotExist(err) {
		if err := refreshFriendsCache(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func ensureGroupsCache() error {
	if _, err := os.Stat(filepath.Join(CacheDir, "groups.json")); os.IsNotExist(err) {
		if err := refreshGroupsCache(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func refreshFriendsCache() error {
	client, err := splitwise.NewClient()
	if err != nil {
		return fmt.Errorf("initializing client: %w", err)
	}
	data, err := client.GetFriends()
	if err != nil {
		return fmt.Errorf("fetching friends: %w", err)
	}
	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(CacheDir, "friends.json"), data, 0644); err != nil {
		return fmt.Errorf("writing friends cache: %w", err)
	}
	return nil
}

func refreshGroupsCache() error {
	client, err := splitwise.NewClient()
	if err != nil {
		return fmt.Errorf("initializing client: %w", err)
	}
	data, err := client.GetGroups()
	if err != nil {
		return fmt.Errorf("fetching groups: %w", err)
	}
	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(CacheDir, "groups.json"), data, 0644); err != nil {
		return fmt.Errorf("writing groups cache: %w", err)
	}
	return nil
}

func initializeNewExpense(groupID, friendID int) (*splitwise.DetailedExpense, error) {
	if _, err := os.Stat(filepath.Join(CacheDir, "current_user.json")); os.IsNotExist(err) {
		if err := fetchCurrentUser(); err != nil {
			return nil, err
		}
	}
	currentUser, err := splitwise.GetCachedCurrentUser(CacheDir)
	if err != nil {
		return nil, fmt.Errorf("reading cached current user: %w", err)
	}

	expense := &splitwise.DetailedExpense{
		Description:  "",
		Cost:         "0.00",
		CurrencyCode: currentUser.DefaultCurrency,
		Date:         time.Now().Format("2006-01-02"),
		GroupID:      groupID,
	}
	if expense.CurrencyCode == "" {
		expense.CurrencyCode = "AUD"
	}

	addUser := func(id int, firstName, lastName string) {
		var lastNamePtr *string
		if lastName != "" {
			lastNameValue := lastName
			lastNamePtr = &lastNameValue
		}
		expense.Users = append(expense.Users, splitwise.ExpenseUser{
			UserID: id,
			User: splitwise.User{
				ID:        id,
				FirstName: firstName,
				LastName:  lastNamePtr,
			},
			PaidShare: "0.00",
			OwedShare: "0.00",
		})
	}

	addUser(currentUser.ID, currentUser.FirstName, currentUser.LastName)

	if friendID != 0 {
		if err := ensureFriendsCache(); err != nil {
			return nil, err
		}
		friends, err := splitwise.GetCachedFriends(CacheDir)
		if err != nil {
			return nil, err
		}
		for _, f := range friends {
			if f.ID == friendID {
				addUser(f.ID, f.FirstName, f.LastName)
				return expense, nil
			}
		}
		return nil, fmt.Errorf("friend %d not found", friendID)
	}

	if groupID != 0 {
		if err := ensureGroupsCache(); err != nil {
			return nil, err
		}
		groups, err := splitwise.GetCachedGroups(CacheDir)
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			if g.ID != groupID {
				continue
			}
			expense.Description = g.Name
			foundCurrent := false
			expense.Users = expense.Users[:0]
			for _, m := range g.Members {
				if m.ID == currentUser.ID {
					foundCurrent = true
				}
				addUser(m.ID, m.FirstName, m.LastName)
			}
			if !foundCurrent {
				expense.Users = append([]splitwise.ExpenseUser{{
					UserID: currentUser.ID,
					User: splitwise.User{
						ID:        currentUser.ID,
						FirstName: currentUser.FirstName,
						LastName:  stringPtrOrNil(currentUser.LastName),
					},
					PaidShare: "0.00",
					OwedShare: "0.00",
				}}, expense.Users...)
			}
			return expense, nil
		}
		return nil, fmt.Errorf("group %d not found", groupID)
	}

	return expense, nil
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	v := s
	return &v
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
