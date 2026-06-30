// Package gemini implements the llm.Provider interface for Google Gemini.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"github.com/yjlee/aibrow/src/llm"
	"google.golang.org/api/option"
)

// newGenaiClient is a seam for tests to inject an alternate client constructor.
var newGenaiClient = func(ctx context.Context, opts ...option.ClientOption) (*genai.Client, error) {
	return genai.NewClient(ctx, opts...)
}

// schemaInput mirrors a subset of the OpenAPI/JSON Schema format that Gemini
// accepts for function declaration parameters.
type schemaInput struct {
	Type        string                     `json:"type"`
	Format      string                     `json:"format"`
	Description string                     `json:"description"`
	Nullable    bool                       `json:"nullable"`
	Enum        []string                   `json:"enum"`
	Items       json.RawMessage            `json:"items"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
}

// schemaFromJSON converts a JSON Schema payload into a genai.Schema tree.
func schemaFromJSON(raw json.RawMessage) (*genai.Schema, error) {
	var in schemaInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	out := &genai.Schema{
		Type:        schemaType(in.Type),
		Format:      in.Format,
		Description: in.Description,
		Nullable:    in.Nullable,
		Enum:        in.Enum,
		Required:    in.Required,
	}
	if len(in.Properties) > 0 {
		out.Properties = make(map[string]*genai.Schema)
		for name, childRaw := range in.Properties {
			child, err := schemaFromJSON(childRaw)
			if err != nil {
				return nil, err
			}
			out.Properties[name] = child
		}
	}
	if len(in.Items) > 0 {
		child, err := schemaFromJSON(in.Items)
		if err != nil {
			return nil, err
		}
		out.Items = child
	}
	return out, nil
}

func schemaType(s string) genai.Type {
	switch s {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeUnspecified
	}
}

// Provider is a Gemini-backed llm.Provider.
type Provider struct {
	client *genai.Client
	model  string
}

// NewProvider creates a Gemini Provider. apiKey is required.
func NewProvider(ctx context.Context, apiKey, model string) (*Provider, error) {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	client, err := newGenaiClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	return &Provider{client: client, model: model}, nil
}

// Chat implements llm.Provider.
func (p *Provider) Chat(ctx context.Context, messages []llm.Message, tools []llm.Tool) (string, []llm.ToolCall, error) {
	model := p.client.GenerativeModel(p.model)

	for _, t := range tools {
		decl := &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
		}
		if len(t.Parameters) > 0 {
			if schema, err := schemaFromJSON(t.Parameters); err == nil {
				decl.Parameters = schema
			}
		}
		model.Tools = append(model.Tools, &genai.Tool{FunctionDeclarations: []*genai.FunctionDeclaration{decl}})
	}

	chat := model.StartChat()
	for _, m := range messages {
		content := &genai.Content{Role: string(m.Role), Parts: []genai.Part{genai.Text(m.Content)}}
		chat.History = append(chat.History, content)
	}

	resp, err := chat.SendMessage(ctx, genai.Text(""))
	if err != nil {
		return "", nil, fmt.Errorf("gemini chat: %w", err)
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", nil, fmt.Errorf("gemini: no candidates returned")
	}

	var text string
	var calls []llm.ToolCall
	for _, part := range resp.Candidates[0].Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			text += string(v)
		case genai.FunctionCall:
			args, _ := json.Marshal(v.Args)
			calls = append(calls, llm.ToolCall{Name: v.Name, Arguments: args})
		}
	}
	return text, calls, nil
}
