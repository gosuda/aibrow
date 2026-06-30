// Package openai implements the llm.Provider interface for OpenAI-compatible APIs.
package openai

import (
	"context"
	"encoding/json"
	"fmt"

	oai "github.com/sashabaranov/go-openai"
	"github.com/yjlee/aibrow/src/llm"
)

// Provider is an OpenAI-backed llm.Provider.
type Provider struct {
	client *oai.Client
	model  string
}

// NewProvider creates a Provider using the given API key, base URL, and model.
// An empty base URL uses the default OpenAI endpoint.
func NewProvider(apiKey, baseURL, model string) *Provider {
	config := oai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	if model == "" {
		model = oai.GPT4o
	}
	return &Provider{client: oai.NewClientWithConfig(config), model: model}
}

// Chat implements llm.Provider.
func (p *Provider) Chat(ctx context.Context, messages []llm.Message, tools []llm.Tool) (string, []llm.ToolCall, error) {
	msgs := make([]oai.ChatCompletionMessage, 0, len(messages))
	for _, m := range messages {
		msgs = append(msgs, oai.ChatCompletionMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	var oaiTools []oai.Tool
	for _, t := range tools {
		oaiTools = append(oaiTools, oai.Tool{
			Type: "function",
			Function: &oai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	req := oai.ChatCompletionRequest{
		Model:    p.model,
		Messages: msgs,
		Tools:    oaiTools,
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("openai chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("openai: no choices returned")
	}

	choice := resp.Choices[0].Message
	var calls []llm.ToolCall
	for _, tc := range choice.ToolCalls {
		if tc.Function.Name == "" {
			continue
		}
		calls = append(calls, llm.ToolCall{
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}
	return choice.Content, calls, nil
}
