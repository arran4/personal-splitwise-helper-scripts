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
const GroupsCacheFile = "groups.json"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: groups <get|list>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "get":
		if err := fetchGroups(); err != nil {
			fmt.Println("Error fetching groups:", err)
			os.Exit(1)
		}
	case "list":
		if err := listGroups(); err != nil {
			fmt.Println("Error listing groups:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func fetchGroups() error {
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

	cachePath := filepath.Join(CacheDir, GroupsCacheFile)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("writing to cache file: %w", err)
	}

	fmt.Printf("Successfully fetched groups and cached to %s\n", cachePath)
	return nil
}

func listGroups() error {
	cachePath := filepath.Join(CacheDir, GroupsCacheFile)

	fileInfo, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		fmt.Println("Cache file not found. Fetching groups...")
		if err := fetchGroups(); err != nil {
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

	var resp splitwise.GroupsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing groups JSON: %w", err)
	}

	fmt.Printf("Cache last modified: %s\n\n", fileInfo.ModTime().Format(time.RFC1123))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tMEMBERS")
	for _, g := range resp.Groups {
		var members []string
		for _, m := range g.Members {
			name := strings.TrimSpace(strings.Join([]string{m.FirstName, m.LastName}, " "))
			if name == "" {
				name = m.Email
			}
			members = append(members, name)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\n", g.ID, g.Name, strings.Join(members, ", "))
	}
	w.Flush()

	return nil
}
