package splitwise

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type CurrentUserResponse struct {
	User CurrentUser `json:"user"`
}

type CurrentUser struct {
	ID                 int    `json:"id"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Email              string `json:"email"`
	RegistrationStatus string `json:"registration_status"`
	DefaultCurrency    string `json:"default_currency"`
	Locale             string `json:"locale"`
}

func GetCachedCurrentUser(cacheDir string) (*CurrentUser, error) {
	cachePath := filepath.Join(cacheDir, "current_user.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var resp CurrentUserResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return &resp.User, nil
}
