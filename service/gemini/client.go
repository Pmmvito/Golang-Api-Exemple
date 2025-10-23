package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultModel      = "gemini-2.5-flash-preview-05-20"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	defaultBackoff    = time.Second
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	model      string
	maxRetries int
	backoff    time.Duration
}

type Option func(*Client)

func WithModel(model string) Option {
	return func(c *Client) {
		if model != "" {
			c.model = model
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithRetry(maxRetries int, backoff time.Duration) Option {
	return func(c *Client) {
		if maxRetries > 0 {
			c.maxRetries = maxRetries
		}
		if backoff > 0 {
			c.backoff = backoff
		}
	}
}

func NewClientFromEnv(opts ...Option) (*Client, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("EXPO_PUBLIC_GEMINI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key não configurada. defina GEMINI_API_KEY ou EXPO_PUBLIC_GEMINI_API_KEY")
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = os.Getenv("EXPO_PUBLIC_GEMINI_MODEL")
	}

	client := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		apiKey:     apiKey,
		model:      defaultModel,
		maxRetries: defaultMaxRetries,
		backoff:    defaultBackoff,
	}

	if model != "" {
		client.model = model
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

type ContentPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type Content struct {
	Role  string        `json:"role,omitempty"`
	Parts []ContentPart `json:"parts"`
}

type GenerateContentRequest struct {
	Contents []Content `json:"contents"`
}

type Candidate struct {
	Content CandidateContent `json:"content"`
}

type CandidateContent struct {
	Parts []ContentPart `json:"parts"`
}

type UsageMetadata struct {
	PromptTokenCount     int64 `json:"promptTokenCount"`
	CandidatesTokenCount int64 `json:"candidatesTokenCount"`
	TotalTokenCount      int64 `json:"totalTokenCount"`
}

type generateContentResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

type Result struct {
	Text  string
	Usage UsageMetadata
}

func (c *Client) GenerateContent(ctx context.Context, payload GenerateContentRequest) (*Result, error) {
	if len(payload.Contents) == 0 {
		return nil, fmt.Errorf("payload inválido: contents vazio")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro serializando payload gemini: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	var lastErr error
	var result *Result
	backoff := c.backoff

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("erro criando request gemini: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			func() {
				defer resp.Body.Close()
				if resp.StatusCode < 200 || resp.StatusCode >= 300 {
					lastErr = fmt.Errorf("resposta gemini inválida: status %d", resp.StatusCode)
					return
				}
				var apiResp generateContentResponse
				if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
					lastErr = fmt.Errorf("erro decodificando resposta gemini: %w", err)
					return
				}
				text := extractText(apiResp.Candidates)
				if text == "" {
					lastErr = fmt.Errorf("resposta da api gemini sem texto utilizável")
					return
				}
				result = &Result{
					Text:  strings.TrimSpace(text),
					Usage: apiResp.UsageMetadata,
				}
			}()
			if result != nil {
				return result, nil
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("falha desconhecida na comunicação com gemini")
	}

	return nil, lastErr
}

func extractText(candidates []Candidate) string {
	if len(candidates) == 0 {
		return ""
	}
	for _, candidate := range candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				return part.Text
			}
		}
	}
	return ""
}

func SanitizeJSON(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```JSON")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}

func NewTextPart(text string) ContentPart {
	return ContentPart{Text: text}
}

func NewInlineImagePart(mimeType, data string) ContentPart {
	return ContentPart{
		InlineData: &InlineData{
			MimeType: mimeType,
			Data:     data,
		},
	}
}
