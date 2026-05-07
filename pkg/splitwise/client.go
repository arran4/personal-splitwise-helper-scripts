package splitwise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/arran4/personal-splitwise-helper-scripts/pkg/config"
)

const BaseURL = "https://secure.splitwise.com/api/v3.0"

type Client struct {
	token string
}

func NewClient() (*Client, error) {
	token, err := config.ReadToken()
	if err != nil {
		return nil, err
	}
	return &Client{token: token}, nil
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, BaseURL+endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetFriends() ([]byte, error) {
	return c.doRequest("GET", "/get_friends", nil)
}

func (c *Client) GetGroups() ([]byte, error) {
	return c.doRequest("GET", "/get_groups", nil)
}

func (c *Client) GetExpenses(query string) ([]byte, error) {
	endpoint := "/get_expenses"
	if query != "" {
		endpoint += "?" + query
	}
	return c.doRequest("GET", endpoint, nil)
}

func (c *Client) GetExpense(id string) ([]byte, error) {
	return c.doRequest("GET", fmt.Sprintf("/get_expense/%s", id), nil)
}

func (c *Client) GetCurrentUser() ([]byte, error) {
	return c.doRequest("GET", "/get_current_user", nil)
}

func (c *Client) UpdateExpense(expense *DetailedExpense) ([]byte, error) {
	if expense == nil {
		return nil, fmt.Errorf("expense is nil")
	}

	payload := buildExpensePayload(expense, true)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.doRequest("POST", fmt.Sprintf("/update_expense/%d", expense.ID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Errors interface{} `json:"errors"`
	}
	if err := json.Unmarshal(data, &response); err == nil && hasAPIError(response.Errors) {
		return nil, fmt.Errorf("Splitwise API returned errors: %v", response.Errors)
	}

	return data, nil
}

func (c *Client) CreateExpense(expense *DetailedExpense) ([]byte, error) {
	if expense == nil {
		return nil, fmt.Errorf("expense is nil")
	}

	payload := buildExpensePayload(expense, false)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.doRequest("POST", "/create_expense", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Errors interface{} `json:"errors"`
	}
	if err := json.Unmarshal(data, &response); err == nil && hasAPIError(response.Errors) {
		return nil, fmt.Errorf("Splitwise API returned errors: %v", response.Errors)
	}

	return data, nil
}

func buildExpensePayload(expense *DetailedExpense, includeGroupZero bool) map[string]interface{} {
	payload := map[string]interface{}{
		"cost":          expense.Cost,
		"description":   expense.Description,
		"details":       expense.Details,
		"date":          expense.Date,
		"currency_code": expense.CurrencyCode,
	}
	if groupID := normalizeGroupID(expense.GroupID); groupID > 0 || includeGroupZero {
		payload["group_id"] = groupID
	}
	if expense.Category.ID != 0 {
		payload["category_id"] = expense.Category.ID
	}
	for i, user := range expense.Users {
		payload[fmt.Sprintf("users__%d__user_id", i)] = user.UserID
		payload[fmt.Sprintf("users__%d__paid_share", i)] = user.PaidShare
		payload[fmt.Sprintf("users__%d__owed_share", i)] = user.OwedShare
	}
	return payload
}

func normalizeGroupID(groupID interface{}) int {
	switch v := groupID.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func hasAPIError(v interface{}) bool {
	switch e := v.(type) {
	case nil:
		return false
	case string:
		return e != ""
	case []interface{}:
		return len(e) > 0
	case map[string]interface{}:
		return len(e) > 0
	default:
		return true
	}
}
