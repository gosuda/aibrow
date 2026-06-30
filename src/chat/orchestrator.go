// Package chat implements the composite LLM chat pipeline. It orchestrates a
// provider with the MCP-style tool registry, executing any tool calls requested
// by the model and feeding the results back for follow-up responses.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/tools"
)

// maxIterations bounds the number of tool-call rounds in a single request.
const maxIterations = 5

// ToolCallResult captures the outcome of one tool execution.
type ToolCallResult struct {
	Name    string          `json:"name"`
	Input   json.RawMessage `json:"input"`
	Output  interface{}     `json:"output,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Response is the final result of the chat pipeline.
type Response struct {
	Message   string           `json:"message"`
	ToolCalls []llm.ToolCall   `json:"tool_calls,omitempty"`
	Results   []ToolCallResult `json:"results,omitempty"`
}

// Run drives a chat turn with tool execution. It repeatedly calls the provider
// until the model returns no tool calls or the iteration limit is reached.
func Run(ctx context.Context, provider llm.Provider, registry *tools.Registry, messages []llm.Message) (*Response, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider not configured")
	}
	if registry == nil {
		return nil, fmt.Errorf("tool registry not configured")
	}

	llmTools := buildLLMTools(registry)
	var allCalls []llm.ToolCall
	var results []ToolCallResult

	for i := 0; i < maxIterations; i++ {
		text, calls, err := provider.Chat(ctx, messages, llmTools)
		if err != nil {
			return nil, fmt.Errorf("provider chat: %w", err)
		}
		if len(calls) == 0 {
			return &Response{Message: text, ToolCalls: allCalls, Results: results}, nil
		}

		allCalls = append(allCalls, calls...)
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, Content: text})

		for _, call := range calls {
			output, execErr := registry.Execute(ctx, call.Name, call.Arguments)
			res := ToolCallResult{Name: call.Name, Input: call.Arguments, Output: output}
			content := ""
			if execErr != nil {
				res.Error = execErr.Error()
				content = fmt.Sprintf("Error: %s", execErr.Error())
			} else {
				outBytes, _ := json.Marshal(output)
				content = string(outBytes)
			}
			results = append(results, res)
			messages = append(messages, llm.ToolResultMessage(llm.ToolResult{Name: call.Name, Content: content}))
		}
	}

	return nil, fmt.Errorf("max tool iterations (%d) reached", maxIterations)
}

// buildLLMTools converts the MCP registry into the provider-agnostic tool
// format, sorted by name for deterministic requests.
func buildLLMTools(registry *tools.Registry) []llm.Tool {
	names := registry.Names()
	sort.Strings(names)
	llmTools := make([]llm.Tool, 0, len(names))
	for _, name := range names {
		t, _ := registry.Get(name)
		llmTools = append(llmTools, llm.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		})
	}
	return llmTools
}
