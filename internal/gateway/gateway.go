package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// DiscoverGateway tries to find a running OpenClaw gateway.
// Priority: flagURL > configURL > default port > common ports scan.
func DiscoverGateway(flagURL, configURL string) (string, error) {
	// 1. CLI flag
	if flagURL != "" {
		return flagURL, nil
	}

	// 2. Config file
	if configURL != "" {
		return configURL, nil
	}

	// 3. Auto-detect: OpenClaw default port
	defaultURL := "http://127.0.0.1:18789"
	if pingGateway(defaultURL) {
		return defaultURL, nil
	}

	// 4. Fallback: try common ports
	commonPorts := []int{3000, 8080, 3260}
	for _, port := range commonPorts {
		candidate := fmt.Sprintf("http://127.0.0.1:%d", port)
		if pingGateway(candidate) {
			return candidate, nil
		}
	}

	// 5. Nothing found
	return "", fmt.Errorf("OpenClaw gateway not found. Is it running? (openclaw gateway status)")
}

// DiscoverAuthToken resolves the gateway auth token.
// Priority: OPENCLAW_TOKEN env var > ~/.openclaw/openclaw.json gateway.auth.token > empty.
func DiscoverAuthToken() string {
	if token := os.Getenv("OPENCLAW_TOKEN"); token != "" {
		return token
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".openclaw", "openclaw.json"))
	if err != nil {
		return ""
	}

	var cfg struct {
		Gateway struct {
			Auth struct {
				Token string `json:"token"`
			} `json:"auth"`
		} `json:"gateway"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.Gateway.Auth.Token
}

// pingGateway checks if a gateway is responding at the given URL.
func pingGateway(baseURL string) bool {
	client := &http.Client{Timeout: 2 * time.Second}

	for _, path := range []string{"/api/ping", "/api/status"} {
		resp, err := client.Get(baseURL + path)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return true
		}
	}
	return false
}

// SendMessage sends a message to a thread via /tools/invoke
type ToolInvokeRequest struct {
	Tool       string                 `json:"tool"`
	Args       map[string]interface{} `json:"args"`
	SessionKey string                 `json:"sessionKey,omitempty"`
}

type SendResponse struct {
	OK        bool   `json:"ok"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	ID        string `json:"id,omitempty"`
}

func (c *Client) SendMessage(threadID, message, model, thinking, groupName, presetName string, timeout time.Duration) (*SendResponse, error) {
	// Prepend orchestration context so threads know they are being managed
	var header string
	if groupName != "" || presetName != "" || model != "" {
		parts := []string{}
		if groupName != "" {
			parts = append(parts, "group: "+groupName)
		}
		if presetName != "" {
			parts = append(parts, "preset: "+presetName)
		}
		if model != "" {
			parts = append(parts, "model: "+model)
		}
		header = "**[clawrus]** " + strings.Join(parts, " · ") + "\n"
	}

	message = header + message

	reqBody := ToolInvokeRequest{
		Tool:       "message",
		SessionKey: "channel:" + threadID,
		Args: map[string]interface{}{
			"action":  "send",
			"channel": "discord",
			"target":  threadID,
			"message": message,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/tools/invoke", c.BaseURL)
	httpClient := &http.Client{Timeout: timeout}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway error (%d): %s", resp.StatusCode, string(body))
	}

	var result SendResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("gateway returned non-JSON: %s", string(body))
	}
	if result.Error != "" {
		return nil, fmt.Errorf("gateway: %s", result.Error)
	}
	// Normalize message ID: prefer "messageId", fall back to "id"
	if result.MessageID == "" && result.ID != "" {
		result.MessageID = result.ID
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

// toolInvokeResponse matches the /tools/invoke response envelope for message reads.
type toolInvokeResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		Content json.RawMessage `json:"content"`
		Details struct {
			OK       bool               `json:"ok"`
			Messages []toolInvokeMessage `json:"messages"`
		} `json:"details"`
	} `json:"result"`
}

type toolInvokeMessage struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Author  struct {
		Bot bool `json:"bot"`
	} `json:"author"`
	TimestampMs int64 `json:"timestampMs"`
}

// PollReply polls for new messages in a thread after a given message ID.
// Returns the first non-bot reply content found, or empty string on timeout.
func (c *Client) PollReply(threadID, afterMessageID string, gatherTimeout time.Duration) (string, error) {
	deadline := time.Now().Add(gatherTimeout)
	httpClient := &http.Client{Timeout: 10 * time.Second}

	var afterID uint64
	if afterMessageID != "" {
		var err error
		afterID, err = strconv.ParseUint(afterMessageID, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid afterMessageID %q: %w", afterMessageID, err)
		}
	}

	for time.Now().Before(deadline) {
		reqBody := ToolInvokeRequest{
			Tool: "message",
			Args: map[string]interface{}{
				"action":  "read",
				"channel": "discord",
				"target":  threadID,
				"limit":   10,
			},
		}

		data, err := json.Marshal(reqBody)
		if err != nil {
			return "", err
		}

		url := fmt.Sprintf("%s/tools/invoke", c.BaseURL)
		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		if c.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.APIKey)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			time.Sleep(3 * time.Second)
			continue
		}

		var envelope toolInvokeResponse
		if err := json.Unmarshal(body, &envelope); err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		// Messages are returned newest-first; find the first message after afterID.
		for _, msg := range envelope.Result.Details.Messages {
			if afterID > 0 && msg.ID != "" {
				msgID, err := strconv.ParseUint(msg.ID, 10, 64)
				if err != nil || msgID <= afterID {
					continue
				}
			}
			if msg.Content != "" {
				return msg.Content, nil
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", nil
}

// SummarizeReplies sends replies to OpenClaw gateway's /api/ai/complete for LLM summarization.
// Returns empty string if the endpoint is not available (404).
func SummarizeReplies(gatewayURL, replies string) (string, error) {
	reqBody := map[string]interface{}{
		"prompt": "Summarize these agent replies into a concise unified status:\n" + replies,
		"model":  "glm-5-turbo",
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := gatewayURL + "/api/ai/complete"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", nil // gateway unreachable, degrade gracefully
	}
	defer resp.Body.Close()

	// 404 means /api/ai/complete not available — not an error, just skip summarization
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil // non-JSON response, degrade gracefully
	}

	// Extract content from response
	if content, ok := result["content"].(string); ok {
		return content, nil
	}
	if content, ok := result["text"].(string); ok {
		return content, nil
	}

	// Try OpenAI-compatible format
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", nil
}
