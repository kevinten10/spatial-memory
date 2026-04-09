package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/spatial-memory/spatial-memory/internal/config"
	"github.com/spatial-memory/spatial-memory/internal/model"
)

const (
	defaultBaseURL = "https://open.bigmodel.cn/api/paas/v4"
	defaultTimeout = 30 * time.Second
)

// GLMClient is the interface for GLM-4V moderation client.
type GLMClient interface {
	ModerateImage(ctx context.Context, imageURL string) (*model.ModerationResult, error)
	ModerateText(ctx context.Context, text string) (*model.ModerationResult, error)
}

// Client implements GLMClient using ZhipuAI REST API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new GLM-4V moderation client.
func NewClient(cfg config.GLMConfig) GLMClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// chatRequest represents the request body for GLM-4V API.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// textContent represents text content for the message.
type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// imageContent represents image content for the message.
type imageContent struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

// chatResponse represents the response from GLM-4V API.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ModerateImage moderates an image using GLM-4V.
func (c *Client) ModerateImage(ctx context.Context, imageURL string) (*model.ModerationResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("GLM API key not configured")
	}

	prompt := `Analyze this image for content moderation. Check for:
1. Violence, gore, or dangerous content
2. Hate symbols or extremist content
3. Sexual or adult content
4. Harassment or bullying
5. Illegal activities
6. Spam or misleading content

Respond in JSON format only:
{
  "safe": true/false,
  "confidence": 0.0-1.0,
  "categories": ["category1", "category2"]
}

If safe, categories should be empty. If unsafe, list applicable categories.`

	reqBody := chatRequest{
		Model: "glm-4v",
		Messages: []message{
			{
				Role: "user",
				Content: []interface{}{
					textContent{Type: "text", Text: prompt},
					imageContent{
						Type: "image_url",
						ImageURL: struct {
							URL string `json:"url"`
						}{URL: imageURL},
					},
				},
			},
		},
	}

	return c.doRequestWithRetry(ctx, reqBody)
}

// ModerateText moderates text content using GLM-4V.
func (c *Client) ModerateText(ctx context.Context, text string) (*model.ModerationResult, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("GLM API key not configured")
	}

	prompt := fmt.Sprintf(`Analyze the following text for content moderation:
"%s"

Check for:
1. Hate speech or discrimination
2. Harassment or threats
3. Sexual content
4. Violence or dangerous content
5. Illegal activities
6. Spam or scams
7. Misinformation

Respond in JSON format only:
{
  "safe": true/false,
  "confidence": 0.0-1.0,
  "categories": ["category1", "category2"]
}

If safe, categories should be empty. If unsafe, list applicable categories.`, text)

	reqBody := chatRequest{
		Model: "glm-4",
		Messages: []message{
			{
				Role: "user",
				Content: []textContent{
					{Type: "text", Text: prompt},
				},
			},
		},
	}

	return c.doRequestWithRetry(ctx, reqBody)
}

// doRequestWithRetry performs the API request with 1 retry on 5xx errors.
func (c *Client) doRequestWithRetry(ctx context.Context, reqBody chatRequest) (*model.ModerationResult, error) {
	result, statusCode, err := c.doRequest(ctx, reqBody)
	if err != nil {
		// Retry on 5xx errors
		if statusCode >= 500 && statusCode < 600 {
			log.Warn().Int("status_code", statusCode).Msg("GLM API returned 5xx, retrying once")
			time.Sleep(1 * time.Second)
			result, _, err = c.doRequest(ctx, reqBody)
		}
	}
	return result, err
}

// doRequest performs a single API request.
func (c *Client) doRequest(ctx context.Context, reqBody chatRequest) (*model.ModerationResult, int, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("GLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, resp.StatusCode, fmt.Errorf("GLM API error: %s - %s", chatResp.Error.Code, chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, resp.StatusCode, fmt.Errorf("no choices in GLM response")
	}

	// Parse the content which should be JSON
	content := chatResp.Choices[0].Message.Content
	result, err := parseModerationResult(content)
	if err != nil {
		log.Warn().Err(err).Str("content", content).Msg("failed to parse moderation result, using fallback")
		// Fallback: assume safe if parsing fails
		return &model.ModerationResult{
			Safe:       true,
			Confidence: 0.5,
			Categories: []string{},
		}, resp.StatusCode, nil
	}

	return result, resp.StatusCode, nil
}

// parseModerationResult parses the JSON response from GLM.
func parseModerationResult(content string) (*model.ModerationResult, error) {
	// Try to extract JSON from the response (it might be wrapped in markdown code blocks)
	jsonStr := extractJSON(content)

	var result model.ModerationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse moderation result: %w", err)
	}

	return &result, nil
}

// extractJSON extracts JSON from a string that might be wrapped in markdown.
func extractJSON(content string) string {
	// Look for JSON between triple backticks
	start := 0
	end := len(content)

	// Check for ```json or ``` at the start
	if idx := bytes.Index([]byte(content), []byte("```json")); idx != -1 {
		start = idx + 7
	} else if idx := bytes.Index([]byte(content), []byte("```")); idx != -1 {
		start = idx + 3
	}

	// Check for ``` at the end
	if idx := bytes.LastIndex([]byte(content), []byte("```")); idx != -1 && idx > start {
		end = idx
	}

	return string(bytes.TrimSpace([]byte(content[start:end])))
}
