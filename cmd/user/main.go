package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
)

const CacheDir = ".cache"
const CurrentUserCacheFile = "current_user.json"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: user <get|show>")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "get":
		if err := fetchCurrentUser(); err != nil {
			fmt.Println("Error fetching current user:", err)
			os.Exit(1)
		}
	case "show":
		if err := showCurrentUser(); err != nil {
			fmt.Println("Error showing current user:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
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

	cachePath := filepath.Join(CacheDir, CurrentUserCacheFile)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing to cache file: %w", err)
	}

	fmt.Printf("Successfully fetched current user and cached to %s\n", cachePath)
	return nil
}

func showCurrentUser() error {
	cachePath := filepath.Join(CacheDir, CurrentUserCacheFile)

	fileInfo, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		fmt.Println("Cache file not found. Fetching current user...")
		if err := fetchCurrentUser(); err != nil {
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

	var resp splitwise.CurrentUserResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing current user JSON: %w", err)
	}

	fmt.Printf("Cache last modified: %s\n\n", fileInfo.ModTime().Format(time.RFC1123))

	u := resp.User
	fmt.Printf("ID: %d\n", u.ID)
	fmt.Printf("Name: %s %s\n", u.FirstName, u.LastName)
	fmt.Printf("Email: %s\n", u.Email)
	fmt.Printf("Default Currency: %s\n", u.DefaultCurrency)
	fmt.Printf("Locale: %s\n", u.Locale)

	return nil
}
