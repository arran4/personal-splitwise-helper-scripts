package splitwise

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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

type GroupsResponse struct {
	Groups []Group `json:"groups"`
}

type Group struct {
	ID      int         `json:"id"`
	Name    string      `json:"name"`
	Members []GroupUser `json:"members"`
}

type GroupUser struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Balance   []Balance `json:"balance"`
}

func GetCachedFriends(cacheDir string) ([]Friend, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "friends.json"))
	if err != nil {
		return nil, err
	}

	var resp FriendsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return resp.Friends, nil
}

func GetCachedGroups(cacheDir string) ([]Group, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "groups.json"))
	if err != nil {
		return nil, err
	}

	var resp GroupsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return resp.Groups, nil
}
