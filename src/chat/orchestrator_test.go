package chat

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/tools"
)

type fakeProvider struct {
	responses []providerResponse
	calls     int
}

type providerResponse struct {
	text  string
	calls []llm.ToolCall
	err   error
}

func (f *fakeProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.Tool) (string, []llm.ToolCall, error) {
	if f.calls >= len(f.responses) {
		return "", nil, errors.New("no more responses")
	}
	r := f.responses[f.calls]
	f.calls++
	return r.text, r.calls, r.err
}

type fakeExecutor struct {
	output interface{}
	err    error
}

func TestRunProviderNotConfigured(t *testing.T) {
	_, err := Run(context.Background(), nil, tools.NewRegistry(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRegistryNotConfigured(t *testing.T) {
	p := &fakeProvider{}
	_, err := Run(context.Background(), p, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunProviderError(t *testing.T) {
	p := &fakeProvider{responses: []providerResponse{{err: errors.New("chat failed")}}}
	_, err := Run(context.Background(), p, tools.NewRegistry(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunNoToolCalls(t *testing.T) {
	p := &fakeProvider{responses: []providerResponse{{text: "hello"}}}
	resp, err := Run(context.Background(), p, tools.NewRegistry(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Message != "hello" {
		t.Errorf("message = %q", resp.Message)
	}
}

func TestRunWithToolCall(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "greet",
		Description: "greets someone",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var in struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(input, &in); err != nil {
				return nil, err
			}
			return map[string]string{"greeting": "hello " + in.Name}, nil
		},
	})

	p := &fakeProvider{responses: []providerResponse{
		{text: "", calls: []llm.ToolCall{{Name: "greet", Arguments: json.RawMessage(`{"name":"world"}`)}}},
		{text: "done"},
	}}

	resp, err := Run(context.Background(), p, registry, []llm.Message{{Role: llm.RoleUser, Content: "say hi"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Message != "done" {
		t.Errorf("message = %q", resp.Message)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "greet" {
		t.Errorf("tool calls = %v", resp.ToolCalls)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results = %v", resp.Results)
	}
	if resp.Results[0].Error != "" {
		t.Errorf("unexpected error: %s", resp.Results[0].Error)
	}
}

func TestRunToolNotFound(t *testing.T) {
	registry := tools.NewRegistry()
	p := &fakeProvider{responses: []providerResponse{
		{calls: []llm.ToolCall{{Name: "missing", Arguments: json.RawMessage(`{}`)}}},
		{text: "ok"},
	}}

	resp, err := Run(context.Background(), p, registry, []llm.Message{{Role: llm.RoleUser, Content: "test"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Error == "" {
		t.Errorf("expected tool-not-found error, got %v", resp.Results)
	}
}

func TestRunToolExecutionError(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "fail",
		Description: "always fails",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			return nil, errors.New("boom")
		},
	})
	p := &fakeProvider{responses: []providerResponse{
		{calls: []llm.ToolCall{{Name: "fail", Arguments: json.RawMessage(`{}`)}}},
		{text: "noted"},
	}}

	resp, err := Run(context.Background(), p, registry, []llm.Message{{Role: llm.RoleUser, Content: "test"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if resp.Message != "noted" {
		t.Errorf("message = %q", resp.Message)
	}
	if resp.Results[0].Error != "boom" {
		t.Errorf("error = %q", resp.Results[0].Error)
	}
}

func TestRunMaxIterations(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "noop",
		Description: "does nothing",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			return "ok", nil
		},
	})
	p := &fakeProvider{responses: []providerResponse{
		{calls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}}},
		{calls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}}},
		{calls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}}},
		{calls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}}},
		{calls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}}},
		{calls: []llm.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}}},
	}}

	_, err := Run(context.Background(), p, registry, []llm.Message{{Role: llm.RoleUser, Content: "loop"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildLLMToolsSorted(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{Name: "beta", Description: "b"})
	registry.Register(tools.Tool{Name: "alpha", Description: "a"})

	llmTools := buildLLMTools(registry)
	if len(llmTools) != 2 {
		t.Fatalf("len = %d", len(llmTools))
	}
	if llmTools[0].Name != "alpha" || llmTools[1].Name != "beta" {
		t.Errorf("order = %v", llmTools)
	}
}
