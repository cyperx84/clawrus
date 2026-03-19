package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL   string
	Timeout   time.Duration
	APIKey    string
	AgentID   string
	SessionID string // optional, for direct session targeting
}

func NewClient(baseURL, apiKey, agentID string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		AgentID: agentID,
		Timeout: 30 * time.Second,
	}
}

// SendMessage sends a message to a thread via sessions_send
type SendRequest struct {
	Message    string `json:"message"`
	SessionKey string `json:"sessionKey,omitempty"`
	Model      string `json:"model,omitempty"`
	Thinking   string `json:"thinking,omitempty"`
}

type SendResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

func (c *Client) SendMessage(threadID, message, model, thinking string, timeout time.Duration) (*SendResponse, error) {
	reqBody := SendRequest{
		Message: message,
	}
	if model != "" {
		reqBody.Model = model
	}
	if thinking != "" {
		reqBody.Thinking = thinking
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/sessions/send", c.BaseURL)
	httpClient := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if c.AgentID != "" {
		req.Header.Set("X-Agent-ID", c.AgentID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result SendResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway returned non-JSON: %s", string(body))
	}
	return &result, nil
}

// GetStatus fetches session status
func (c *Client) GetStatus(threadID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/sessions/status", c.BaseURL)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	if c.AgentID != "" {
		req.Header.Set("X-Agent-ID", c.AgentID)
	}
	q := req.URL.Query()
	q.Set("sessionKey", threadID)
	req.URL.RawQuery = q.Encode()

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway returned non-JSON: %s", string(body))
	}
	return result, nil
}
