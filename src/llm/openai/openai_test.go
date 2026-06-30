package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	oai "github.com/sashabaranov/go-openai"
	"github.com/yjlee/aibrow/src/llm"
)

func TestNewProviderDefaults(t *testing.T) {
	p := NewProvider("test-key", "", "")
	if p.model != oai.GPT4o {
		t.Errorf("model = %q, want %q", p.model, oai.GPT4o)
	}
	if p.client == nil {
		t.Error("expected client")
	}
}

func TestProviderChat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "hello",
						"tool_calls": []map[string]interface{}{
							{
								"type": "function",
								"function": map[string]interface{}{
									"name":      "test_tool",
									"arguments": `{"x":1}`,
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	p := NewProvider("test-key", ts.URL+"/", "gpt-4o-mini")
	content, calls, err := p.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}}, []llm.Tool{{Name: "test_tool"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "hello" {
		t.Errorf("content = %q, want hello", content)
	}
	if len(calls) != 1 || calls[0].Name != "test_tool" {
		t.Errorf("calls = %v", calls)
	}
}

func TestProviderChatNoChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []map[string]interface{}{}})
	}))
	defer ts.Close()

	p := NewProvider("test-key", ts.URL+"/", "gpt-4o-mini")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error for no choices")
	}
}

func TestProviderChatErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewProvider("test-key", ts.URL+"/", "gpt-4o-mini")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestProviderChatSkipEmptyToolName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":      "assistant",
						"content":   "ok",
						"tool_calls": []map[string]interface{}{
							{
								"type": "function",
								"function": map[string]interface{}{
									"name":      "",
									"arguments": "{}",
								},
							},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	p := NewProvider("test-key", ts.URL+"/", "gpt-4o-mini")
	content, calls, err := p.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "ok" {
		t.Errorf("content = %q", content)
	}
	if len(calls) != 0 {
		t.Errorf("expected empty calls, got %v", calls)
	}
}
