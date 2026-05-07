package main

import (
	"fmt"
	"net/http"
	"os"
	"syscall"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/config"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: token <set|get|delete|test>")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "set":
		fmt.Print("Enter token: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Println("\nError reading token:", err)
			os.Exit(1)
		}
		fmt.Println()
		token := string(bytePassword)
		if err := config.WriteToken(token); err != nil {
			fmt.Println("Error writing token:", err)
			os.Exit(1)
		}
		fmt.Println("Token saved successfully.")
	case "get":
		token, err := config.ReadToken()
		if err != nil {
			fmt.Println("Error reading token:", err)
			os.Exit(1)
		}
		fmt.Println(token)
	case "delete":
		if err := config.DeleteToken(); err != nil {
			fmt.Println("Error deleting token:", err)
			os.Exit(1)
		}
		fmt.Println("Token deleted.")
	case "test":
		token, err := config.ReadToken()
		if err != nil {
			fmt.Println("Error reading token:", err)
			os.Exit(1)
		}

		req, err := http.NewRequest("GET", "https://secure.splitwise.com/api/v3.0/get_current_user", nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			os.Exit(1)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Request failed:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			fmt.Println("Token is valid.")
		} else {
			fmt.Printf("Token test failed with status %d\n", resp.StatusCode)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command:", command)
		os.Exit(1)
	}
}
