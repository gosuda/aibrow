// Package tools implements the MCP-style tool registry used by the LLM
// orchestrator. Each tool declares a JSON input schema and an executor that
// returns structured results.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Executor is the function signature for all tool implementations.
type Executor func(ctx context.Context, input json.RawMessage) (interface{}, error)

// Tool describes one callable tool exposed to LLM providers.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
	Execute     Executor        `json:"-"`
}

// Registry holds registered tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry. It panics if the name is duplicated.
func (r *Registry) Register(t Tool) {
	if _, ok := r.tools[t.Name]; ok {
		panic(fmt.Sprintf("tool %q already registered", t.Name))
	}
	r.tools[t.Name] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Names returns all registered tool names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs the named tool with the provided JSON input.
func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (interface{}, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t.Execute(ctx, input)
}
