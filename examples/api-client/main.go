// Command api-client is a small CLI for the ServerGO HTTP API.
//
// Usage:
//
//	api-client health
//	api-client users
//	api-client create-user <name>
//	api-client telemetry <device-id>
//	api-client command <device-id> <command>
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type client struct {
	baseURL  string
	apiToken string
	http     *http.Client
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	baseURL := os.Getenv("API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	c := &client{
		baseURL:  baseURL,
		apiToken: os.Getenv("API_TOKEN"),
		http:     &http.Client{Timeout: 10 * time.Second},
	}

	switch args[0] {
	case "health":
		return c.do(http.MethodGet, "/health", nil, false)
	case "users":
		return c.do(http.MethodGet, "/api/v1/users", nil, false)
	case "create-user":
		if len(args) != 2 {
			return usageError()
		}
		return c.do(http.MethodPost, "/api/v1/users", map[string]string{"name": args[1]}, false)
	case "telemetry":
		if len(args) != 2 {
			return usageError()
		}
		return c.do(http.MethodGet, devicePath(args[1], "telemetry"), nil, true)
	case "command":
		if len(args) != 3 {
			return usageError()
		}
		return c.do(http.MethodPost, devicePath(args[1], "commands"), map[string]string{"command": args[2]}, true)
	default:
		return usageError()
	}
}

func devicePath(deviceID, resource string) string {
	return "/api/v1/devices/" + url.PathEscape(deviceID) + "/" + resource
}

func (c *client) do(method, path string, body any, auth bool) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		if c.apiToken == "" {
			return errors.New("API_TOKEN must be set")
		}
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") == nil {
		data = pretty.Bytes()
	}
	fmt.Printf("%s\n%s\n", resp.Status, data)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	return nil
}

func usageError() error {
	return errors.New(`usage:
  api-client health
  api-client users
  api-client create-user <name>
  api-client telemetry <device-id>
  api-client command <device-id> <command>

environment: API_URL (default http://localhost:8080), API_TOKEN (device endpoints)`)
}
