// Package llm defines a provider-agnostic interface for chat completions and
// tool calling across OpenAI, Google Gemini, and Ollama backends.
package llm

import (
	"context"
	"encoding/json"
)

// Role identifies the speaker in a conversation turn.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one turn in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Tool describes a callable function exposed to the LLM.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is a request from the model to execute a tool.
type ToolCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult is the outcome of executing a tool.
type ToolResult struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Provider is the unified interface implemented by every LLM driver.
type Provider interface {
	// Chat sends messages plus available tools to the model and returns the
	// assistant message text plus any tool calls requested by the model.
	Chat(ctx context.Context, messages []Message, tools []Tool) (string, []ToolCall, error)
}

// ToolResultMessage builds a user-role message containing a tool result.
func ToolResultMessage(result ToolResult) Message {
	return Message{Role: RoleUser, Content: "Tool result for " + result.Name + ": " + result.Content}
}
