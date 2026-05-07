package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/splitwise"
)

const CacheDir = ".cache"
const FriendsCacheFile = "friends.json"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: friends <get>")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "get":
		client, err := splitwise.NewClient()
		if err != nil {
			fmt.Println("Error initializing client:", err)
			os.Exit(1)
		}

		data, err := client.GetFriends()
		if err != nil {
			fmt.Println("Error fetching friends:", err)
			os.Exit(1)
		}

		if err := os.MkdirAll(CacheDir, 0755); err != nil {
			fmt.Println("Error creating cache directory:", err)
			os.Exit(1)
		}

		cachePath := filepath.Join(CacheDir, FriendsCacheFile)
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			fmt.Println("Error writing to cache file:", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully fetched friends and cached to %s\n", cachePath)
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}
