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
		fmt.Println("Usage: expenses <list|show|edit|new|import> [options]")
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
		pages := listCmd.String("pages", "", "Page selection: N, N-M, N-, or all")

		listCmd.Parse(os.Args[2:])

		allExpenses, err := fetchExpensesPageSet(expenseListOptions{
			groupID:       *groupID,
			friendID:      *friendID,
			datedAfter:    *datedAfter,
			datedBefore:   *datedBefore,
			updatedAfter:  *updatedAfter,
			updatedBefore: *updatedBefore,
			limit:         *limit,
			offset:        *offset,
			pages:         *pages,
		})
		if err != nil {
			fmt.Println("Error fetching expenses:", err)
			os.Exit(1)
		}

		currentUser, _ := getOrFetchCurrentUser()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tDATE\tDESCRIPTION\tRECIPIENT\tCOST")
		for _, e := range allExpenses {
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s %s\n", e.ID, e.Date, e.Description, expenseRecipientSummary(e, currentUser), e.Cost, e.Currency)
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
		limit := editCmd.Int("limit", 20, "Number of recent expenses to fetch per page when selecting")
		offset := editCmd.Int("offset", 0, "Initial offset when selecting a recent expense")
		pages := editCmd.String("pages", "", "Page selection for recent expense chooser: N, N-M, N-, or all")

		editCmd.Parse(os.Args[2:])

		if *id == "" {
			selectedID, err := chooseRecentExpense(expenseListOptions{
				limit:  *limit,
				offset: *offset,
				pages:  *pages,
			})
			if err != nil {
				fmt.Println("Error selecting expense:", err)
				os.Exit(1)
			}
			*id = selectedID
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

	case "import":
		if err := handleImport(os.Args[2:]); err != nil {
			fmt.Println("Error importing expense:", err)
			os.Exit(1)
		}

	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}

type expenseListOptions struct {
	groupID       int
	friendID      int
	datedAfter    string
	datedBefore   string
	updatedAfter  string
	updatedBefore string
	limit         int
	offset        int
	pages         string
}

func fetchExpensesPage(opts expenseListOptions, offset, limit int) ([]splitwise.Expense, error) {
	client, err := splitwise.NewClient()
	if err != nil {
		return nil, fmt.Errorf("initializing client: %w", err)
	}

	params := url.Values{}
	if opts.groupID != 0 {
		params.Add("group_id", strconv.Itoa(opts.groupID))
	}
	if opts.friendID != 0 {
		params.Add("friend_id", strconv.Itoa(opts.friendID))
	}
	if opts.datedAfter != "" {
		params.Add("dated_after", opts.datedAfter)
	}
	if opts.datedBefore != "" {
		params.Add("dated_before", opts.datedBefore)
	}
	if opts.updatedAfter != "" {
		params.Add("updated_after", opts.updatedAfter)
	}
	if opts.updatedBefore != "" {
		params.Add("updated_before", opts.updatedBefore)
	}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	data, err := client.GetExpenses(params.Encode())
	if err != nil {
		return nil, err
	}

	var resp splitwise.ExpensesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing expenses: %w", err)
	}
	return resp.Expenses, nil
}

func fetchExpensesPageSet(opts expenseListOptions) ([]splitwise.Expense, error) {
	var allExpenses []splitwise.Expense
	limit := opts.limit
	if limit <= 0 {
		limit = 20
	}
	pagePlan, err := parsePagesFlag(opts.pages)
	if err != nil {
		return nil, err
	}
	currentOffset := opts.offset
	if pagePlan.startPage > 0 {
		currentOffset = (pagePlan.startPage - 1) * limit
	}
	pagesFetched := 0

	for {
		expenses, err := fetchExpensesPage(opts, currentOffset, limit)
		if err != nil {
			return nil, err
		}

		allExpenses = append(allExpenses, expenses...)
		pagesFetched++

		if len(expenses) < limit {
			break
		}
		if !pagePlan.fetchAll && pagesFetched >= pagePlan.pageCount {
			break
		}
		currentOffset += limit
	}

	return allExpenses, nil
}

type pagePlan struct {
	startPage int
	pageCount int
	fetchAll  bool
}

type recentExpensePageCursor struct {
	limit          int
	nextOffset     int
	remainingPages int
	fetchAll       bool
}

func parsePagesFlag(raw string) (pagePlan, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return pagePlan{startPage: 0, pageCount: 1, fetchAll: false}, nil
	}
	if strings.EqualFold(raw, "all") {
		return pagePlan{startPage: 1, pageCount: 0, fetchAll: true}, nil
	}
	if strings.Contains(raw, "-") {
		parts := strings.SplitN(raw, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || start < 1 {
			return pagePlan{}, fmt.Errorf("invalid --pages value %q", raw)
		}
		endPart := strings.TrimSpace(parts[1])
		if endPart == "" {
			return pagePlan{startPage: start, pageCount: 0, fetchAll: true}, nil
		}
		end, err := strconv.Atoi(endPart)
		if err != nil || end < start {
			return pagePlan{}, fmt.Errorf("invalid --pages value %q", raw)
		}
		return pagePlan{startPage: start, pageCount: (end - start) + 1, fetchAll: false}, nil
	}
	page, err := strconv.Atoi(raw)
	if err != nil || page < 1 {
		return pagePlan{}, fmt.Errorf("invalid --pages value %q", raw)
	}
	return pagePlan{startPage: page, pageCount: 1, fetchAll: false}, nil
}

func newRecentExpensePageCursor(opts expenseListOptions) (recentExpensePageCursor, error) {
	limit := opts.limit
	if limit <= 0 {
		limit = 20
	}
	if strings.TrimSpace(opts.pages) == "" {
		return recentExpensePageCursor{
			limit:          limit,
			nextOffset:     opts.offset,
			remainingPages: 0,
			fetchAll:       true,
		}, nil
	}

	plan, err := parsePagesFlag(opts.pages)
	if err != nil {
		return recentExpensePageCursor{}, err
	}

	nextOffset := opts.offset
	if plan.startPage > 0 {
		nextOffset = (plan.startPage - 1) * limit
	}
	return recentExpensePageCursor{
		limit:          limit,
		nextOffset:     nextOffset,
		remainingPages: plan.pageCount,
		fetchAll:       plan.fetchAll,
	}, nil
}

func (c recentExpensePageCursor) canLoadMore() bool {
	return c.fetchAll || c.remainingPages > 0
}

func (c *recentExpensePageCursor) consumePage(resultCount int) bool {
	if !c.fetchAll && c.remainingPages > 0 {
		c.remainingPages--
	}
	c.nextOffset += c.limit
	if resultCount < c.limit {
		c.fetchAll = false
		c.remainingPages = 0
	}
	return c.canLoadMore()
}

func getOrFetchCurrentUser() (*splitwise.CurrentUser, error) {
	user, err := splitwise.GetCachedCurrentUser(CacheDir)
	if err == nil {
		return user, nil
	}
	if err := fetchCurrentUser(); err != nil {
		return nil, err
	}
	return splitwise.GetCachedCurrentUser(CacheDir)
}

func expenseRecipientSummary(expense splitwise.Expense, currentUser *splitwise.CurrentUser) string {
	if len(expense.Users) == 0 {
		if expense.GroupID != nil {
			return fmt.Sprintf("Group %v", expense.GroupID)
		}
		return "-"
	}

	var recipients []string
	for _, eu := range expense.Users {
		if currentUser != nil && eu.UserID == currentUser.ID {
			continue
		}
		lastName := ""
		if eu.User.LastName != nil {
			lastName = *eu.User.LastName
		}
		name := strings.TrimSpace(fmt.Sprintf("%s %s", eu.User.FirstName, lastName))
		if name != "" {
			recipients = append(recipients, name)
		}
	}
	if len(recipients) == 0 {
		return "-"
	}
	return strings.Join(recipients, ", ")
}

func chooseRecentExpense(opts expenseListOptions) (string, error) {
	return chooseRecentExpenseWithConfig(opts, "Select Recent Expense", "Recent transaction selection")
}

func chooseRecentExpenseWithConfig(opts expenseListOptions, title, footerLabel string) (string, error) {
	cursor, err := newRecentExpensePageCursor(opts)
	if err != nil {
		return "", err
	}

	currentUser, _ := getOrFetchCurrentUser()
	expenses := make([]splitwise.Expense, 0)
	rows := make([]tui.TableSelectionOption, 0)

	loadPage := func() (bool, error) {
		if !cursor.canLoadMore() {
			return false, nil
		}
		pageExpenses, err := fetchExpensesPage(opts, cursor.nextOffset, cursor.limit)
		if err != nil {
			return false, err
		}
		expenses = append(expenses, pageExpenses...)
		for _, expense := range pageExpenses {
			cost := fmt.Sprintf("%s %s", expense.Cost, expense.Currency)
			recipient := expenseRecipientSummary(expense, currentUser)
			rows = append(rows, tui.TableSelectionOption{
				Cells:       []string{strconv.Itoa(expense.ID), expense.Date, expense.Description, recipient, cost},
				FilterValue: strings.Join([]string{strconv.Itoa(expense.ID), expense.Date, expense.Description, recipient, cost}, " "),
			})
		}
		return cursor.consumePage(len(pageExpenses)), nil
	}

	hasMore, err := loadPage()
	if err != nil {
		return "", err
	}
	if len(expenses) == 0 {
		return "", fmt.Errorf("no expenses found")
	}

	selected, err := tui.SelectTableOption(
		title,
		[]string{"ID", "Date", "Description", "Recipient", "Cost"},
		rows,
		recentExpenseFooter(opts, footerLabel, len(expenses), hasMore),
		hasMore,
		func() ([]tui.TableSelectionOption, string, bool, error) {
			newHasMore, err := loadPage()
			if err != nil {
				return nil, "", false, err
			}
			return rows, recentExpenseFooter(opts, footerLabel, len(expenses), newHasMore), newHasMore, nil
		},
	)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(expenses[selected].ID), nil
}

func recentExpenseFooter(opts expenseListOptions, label string, loaded int, hasMore bool) string {
	mode := fmt.Sprintf("limit=%d offset=%d", opts.limit, opts.offset)
	if strings.TrimSpace(opts.pages) != "" {
		mode += fmt.Sprintf(" pages=%s", opts.pages)
	}
	status := fmt.Sprintf("Loaded %d recent transaction(s)", loaded)
	if hasMore {
		status += " - more available"
	}
	return label + "\n" + status + "\n" + mode
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
