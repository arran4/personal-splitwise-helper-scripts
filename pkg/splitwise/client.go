package splitwise

import (
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
