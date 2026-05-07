package config

import (
	"fmt"
	"os"
	"strings"
)

const TokenFile = ".token.toml"

// ReadToken reads a minimal TOML file format: token = "..."
func ReadToken() (string, error) {
	data, err := os.ReadFile(TokenFile)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "token") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				return val, nil
			}
		}
	}
	return "", fmt.Errorf("token not found in %s", TokenFile)
}

func WriteToken(token string) error {
	content := fmt.Sprintf("token = %q\n", token)
	return os.WriteFile(TokenFile, []byte(content), 0600)
}

func DeleteToken() error {
	return os.Remove(TokenFile)
}
