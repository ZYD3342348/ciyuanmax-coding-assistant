package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://llm.ciyuanmax.art"

var ErrUnauthorized = errors.New("ciyuanmax login expired")

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type APIError struct {
	Status    int
	Code      string `json:"code"`
	ErrorText string `json:"error"`
	Message   string `json:"message"`
}

func (e *APIError) Error() string {
	message := e.Message
	if message == "" {
		message = e.ErrorText
	}
	if message == "" {
		message = http.StatusText(e.Status)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s (%s)", message, e.Code)
	}
	return message
}

type DeviceStart struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DevicePoll struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Me struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	BalanceCNY  string `json:"balance_cny"`
}

type Status struct {
	Email          string            `json:"email"`
	BalanceCNY     string            `json:"balance_cny"`
	MonthSpendCNY  string            `json:"month_spend_cny"`
	AvailableTools []string          `json:"available_tools"`
	CurrentModes   map[string]string `json:"current_modes"`
}

type Mode struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	Recommended bool     `json:"recommended"`
	Available   bool     `json:"available"`
}

type EnsureKeyRequest struct {
	Tool       string `json:"tool"`
	Mode       string `json:"mode"`
	DeviceName string `json:"device_name"`
}

type Key struct {
	KeyID      int64  `json:"key_id"`
	Key        string `json:"key,omitempty"`
	Tool       string `json:"tool,omitempty"`
	Mode       string `json:"mode"`
	ModeTitle  string `json:"mode_title"`
	Endpoint   string `json:"endpoint"`
	Model      string `json:"model"`
	WireAPI    string `json:"wire_api,omitempty"`
	MaskedKey  string `json:"masked_key,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
	Status     string `json:"status,omitempty"`
	LastUsedAt string `json:"last_used_at,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	Current    bool   `json:"current,omitempty"`
}

func NewClient(baseURL, token string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{BaseURL: baseURL, Token: token, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *Client) StartDevice(ctx context.Context) (DeviceStart, error) {
	var result DeviceStart
	err := c.do(ctx, http.MethodPost, "/api/cli/device/start", nil, &result)
	return result, err
}

func (c *Client) PollDevice(ctx context.Context, deviceCode string) (DevicePoll, error) {
	var result DevicePoll
	err := c.do(ctx, http.MethodPost, "/api/cli/device/poll", map[string]string{"device_code": deviceCode}, &result)
	return result, err
}

func (c *Client) Me(ctx context.Context) (Me, error) {
	var result Me
	err := c.do(ctx, http.MethodGet, "/api/cli/me", nil, &result)
	return result, err
}

func (c *Client) Status(ctx context.Context) (Status, error) {
	var result Status
	err := c.do(ctx, http.MethodGet, "/api/cli/status", nil, &result)
	return result, err
}

func (c *Client) Modes(ctx context.Context) ([]Mode, error) {
	var result []Mode
	err := c.do(ctx, http.MethodGet, "/api/cli/modes", nil, &result)
	return result, err
}

func (c *Client) EnsureKey(ctx context.Context, request EnsureKeyRequest) (Key, error) {
	var result Key
	err := c.do(ctx, http.MethodPost, "/api/cli/keys/ensure", request, &result)
	return result, err
}

func (c *Client) ListKeys(ctx context.Context, deviceName string) ([]Key, error) {
	path := "/api/cli/keys"
	if strings.TrimSpace(deviceName) != "" {
		path = "/api/cli/keys/current?" + url.Values{"device_name": []string{deviceName}}.Encode()
	}
	var result []Key
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	return result, err
}

func (c *Client) RotateKey(ctx context.Context, request EnsureKeyRequest) (Key, error) {
	var result Key
	err := c.do(ctx, http.MethodPost, "/api/cli/keys/rotate", request, &result)
	return result, err
}

func (c *Client) RevokeKey(ctx context.Context, keyID int64) error {
	return c.do(ctx, http.MethodPost, "/api/cli/keys/revoke", map[string]int64{"key_id": keyID}, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		request.Header.Set("Authorization", "Bearer "+c.Token)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if response.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError APIError
		_ = json.Unmarshal(data, &apiError)
		apiError.Status = response.StatusCode
		return &apiError
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
