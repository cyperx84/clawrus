package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// PollReply polls for new messages in a thread after a given message ID.
// Returns the first reply content found, or empty string on timeout.
func (c *Client) PollReply(threadID, afterMessageID string, gatherTimeout time.Duration) (string, error) {
	deadline := time.Now().Add(gatherTimeout)
	httpClient := &http.Client{Timeout: 10 * time.Second}

	for time.Now().Before(deadline) {
		url := fmt.Sprintf("%s/api/channels/%s/messages?limit=5&after=%s", c.BaseURL, threadID, afterMessageID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", err
		}
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

		var messages []map[string]interface{}
		if err := json.Unmarshal(body, &messages); err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		if len(messages) > 0 {
			if content, ok := messages[0]["content"].(string); ok && content != "" {
				return content, nil
			}
		}

		time.Sleep(3 * time.Second)
	}

	return "", nil
}

// SummarizeReplies calls an LLM API to aggregate replies.
// Checks OPENROUTER_API_KEY first, then OPENAI_API_KEY. Returns empty string if no key.
func SummarizeReplies(replies string) (string, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	apiURL := "https://openrouter.ai/api/v1/chat/completions"
	model := "openrouter/auto"

	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
		apiURL = "https://api.openai.com/v1/chat/completions"
		model = "gpt-4o-mini"
	}

	if apiKey == "" {
		return "", nil
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": "Summarize these agent replies into a concise unified status:\n" + replies,
			},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("LLM API returned non-JSON: %s", string(body))
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("unexpected LLM response: %s", string(body))
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected choice format")
	}
	msg, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected message format")
	}
	content, _ := msg["content"].(string)
	return content, nil
}
