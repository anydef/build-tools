package resources

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// OPNsenseClient handles HTTP communication with the OPNsense API.
type OPNsenseClient struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
}

// NewOPNsenseClient creates a new API client.
func NewOPNsenseClient(baseURL, apiKey, apiSecret string, insecure bool) *OPNsenseClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,
		},
	}
	return &OPNsenseClient{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		APIKey:    apiKey,
		APISecret: apiSecret,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// doRequest executes an HTTP request with authentication.
func (c *OPNsenseClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.APIKey, c.APISecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	tflog.Debug(ctx, "OPNsense API request", map[string]interface{}{
		"method": method,
		"url":    url,
	})

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	tflog.Debug(ctx, "OPNsense API response", map[string]interface{}{
		"status": resp.StatusCode,
		"body":   string(respBody),
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Create sends a POST to create a resource and returns the UUID.
func (c *OPNsenseClient) Create(ctx context.Context, path string, payload interface{}) (string, error) {
	respBody, err := c.doRequest(ctx, "POST", path, payload)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse create response: %w", err)
	}

	if r, ok := result["result"].(string); ok && r == "failed" {
		return "", fmt.Errorf("create failed: %s", string(respBody))
	}

	uuid, ok := result["uuid"].(string)
	if !ok || uuid == "" {
		return "", fmt.Errorf("no uuid in create response: %s", string(respBody))
	}

	return uuid, nil
}

// Read sends a GET to read a resource and returns the raw JSON body.
func (c *OPNsenseClient) Read(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, "GET", path, nil)
}

// Update sends a POST to update a resource.
func (c *OPNsenseClient) Update(ctx context.Context, path string, payload interface{}) error {
	respBody, err := c.doRequest(ctx, "POST", path, payload)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse update response: %w", err)
	}

	if r, ok := result["result"].(string); ok && r == "failed" {
		return fmt.Errorf("update failed: %s", string(respBody))
	}

	return nil
}

// Delete sends a POST to delete a resource.
func (c *OPNsenseClient) Delete(ctx context.Context, path string) error {
	respBody, err := c.doRequest(ctx, "POST", path, map[string]interface{}{})
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse delete response: %w", err)
	}

	return nil
}

// Post sends a POST request (used for service reconfigure/restart).
func (c *OPNsenseClient) Post(ctx context.Context, path string) error {
	_, err := c.doRequest(ctx, "POST", path, map[string]interface{}{})
	return err
}

// extractStringField safely extracts a string field from a nested map.
// OPNsense API responses may contain arrays, maps, or strings for any field.
// This function handles all cases gracefully.
func extractStringField(data map[string]interface{}, key string) string {
	val, ok := data[key]
	if !ok {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return ""
	}
}

// extractSelectedUUIDs extracts the comma-separated list of selected UUIDs
// from an OPNsense selection field. The API returns these as:
//
//	{"uuid1": {"value": "name1", "selected": 1}, "uuid2": {"value": "name2", "selected": 0}}
//
// This function returns only the UUIDs where selected == 1.
func extractSelectedUUIDs(data map[string]interface{}, key string) string {
	val, ok := data[key]
	if !ok {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case map[string]interface{}:
		var selected []string
		for uuid, entry := range v {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				sel, _ := entryMap["selected"]
				switch s := sel.(type) {
				case float64:
					if s == 1 {
						selected = append(selected, uuid)
					}
				case string:
					if s == "1" {
						selected = append(selected, uuid)
					}
				}
			}
		}
		return strings.Join(selected, ",")
	default:
		return ""
	}
}
