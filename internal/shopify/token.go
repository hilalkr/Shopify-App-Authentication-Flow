package shopify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
}

// temporary authorization code for a permanent access token
func ExchangeCodeForToken(shopDomain, apiKey, apiSecret, code string) (*AccessTokenResponse, error) {
	tokenURL := fmt.Sprintf("https://%s/admin/oauth/access_token", shopDomain)

	requestBody := map[string]string{
		"client_id":     apiKey,
		"client_secret": apiSecret,
		"code":          code,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shopify returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tokenResp, nil
}
