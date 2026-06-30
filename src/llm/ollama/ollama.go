// Package ollama implements the llm.Provider interface for local Ollama servers.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yjlee/aibrow/src/llm"
)

// marshalFn is overridable in tests to simulate JSON marshaling failures.
var marshalFn = json.Marshal

// Provider is an Ollama-backed llm.Provider.
type Provider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewProvider creates an Ollama Provider.
func NewProvider(baseURL, model string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.1"
	}
	return &Provider{baseURL: baseURL, model: model, client: http.DefaultClient}
}

// Chat implements llm.Provider.
func (p *Provider) Chat(ctx context.Context, messages []llm.Message, tools []llm.Tool) (string, []llm.ToolCall, error) {
	ollamaTools := make([]toolSpec, 0, len(tools))
	for _, t := range tools {
		ollamaTools = append(ollamaTools, toolSpec{
			Type:        "function",
			Function:    functionSpec{Name: t.Name, Description: t.Description, Parameters: t.Parameters},
		})
	}

	ollamaMsgs := make([]messageSpec, 0, len(messages))
	for _, m := range messages {
		ollamaMsgs = append(ollamaMsgs, messageSpec{Role: string(m.Role), Content: m.Content})
	}

	payload := chatRequest{Model: p.model, Messages: ollamaMsgs, Tools: ollamaTools, Stream: false}
	body, err := marshalFn(payload)
	if err != nil {
		return "", nil, fmt.Errorf("ollama marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("ollama do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("ollama read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", nil, fmt.Errorf("ollama unmarshal: %w", err)
	}

	var calls []llm.ToolCall
	for _, tc := range chatResp.Message.ToolCalls {
		args, _ := json.Marshal(tc.Function.Arguments)
		calls = append(calls, llm.ToolCall{Name: tc.Function.Name, Arguments: args})
	}
	return chatResp.Message.Content, calls, nil
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []messageSpec  `json:"messages"`
	Tools    []toolSpec     `json:"tools,omitempty"`
	Stream   bool           `json:"stream"`
}

type messageSpec struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toolSpec struct {
	Type     string         `json:"type"`
	Function functionSpec   `json:"function"`
}

type functionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type chatResponse struct {
	Message messageResponse `json:"message"`
}

type messageResponse struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []toolCallSpec   `json:"tool_calls"`
}

type toolCallSpec struct {
	Function struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	} `json:"function"`
}
