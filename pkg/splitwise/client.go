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
