package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
)

const CacheDir = ".cache"
const FriendsCacheFile = "friends.json"

type FriendsResponse struct {
	Friends []Friend `json:"friends"`
}

type Friend struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Balance   []Balance `json:"balance"`
}

type Balance struct {
	CurrencyCode string `json:"currency_code"`
	Amount       string `json:"amount"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: friends <get|list>")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "get":
		if err := fetchFriends(); err != nil {
			fmt.Println("Error fetching friends:", err)
			os.Exit(1)
		}
	case "list":
		if err := listFriends(); err != nil {
			fmt.Println("Error listing friends:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}

func fetchFriends() error {
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

	cachePath := filepath.Join(CacheDir, FriendsCacheFile)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing to cache file: %w", err)
	}

	fmt.Printf("Successfully fetched friends and cached to %s\n", cachePath)
	return nil
}

func listFriends() error {
	cachePath := filepath.Join(CacheDir, FriendsCacheFile)

	fileInfo, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		fmt.Println("Cache file not found. Fetching friends...")
		if err := fetchFriends(); err != nil {
			return err
		}
		fileInfo, err = os.Stat(cachePath)
		if err != nil {
			return fmt.Errorf("getting file info after fetch: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("checking cache file: %w", err)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return fmt.Errorf("reading cache file: %w", err)
	}

	var resp FriendsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing friends JSON: %w", err)
	}

	fmt.Printf("Cache last modified: %s\n\n", fileInfo.ModTime().Format(time.RFC1123))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tFIRST NAME\tLAST NAME\tEMAIL\tBALANCE")
	for _, f := range resp.Friends {
		var balances []string
		for _, b := range f.Balance {
			balances = append(balances, fmt.Sprintf("%s %s", b.Amount, b.CurrencyCode))
		}
		balanceStr := strings.Join(balances, ", ")
		if balanceStr == "" {
			balanceStr = "0.00"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", f.ID, f.FirstName, f.LastName, f.Email, balanceStr)
	}
	w.Flush()

	return nil
}
