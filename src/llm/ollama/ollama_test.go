package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yjlee/aibrow/src/llm"
)

func TestNewProviderDefaults(t *testing.T) {
	p := NewProvider("", "")
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %q", p.baseURL)
	}
	if p.model != "llama3.1" {
		t.Errorf("model = %q", p.model)
	}
}

func TestProviderChat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": "hi there",
				"tool_calls": []map[string]interface{}{
					{
						"function": map[string]interface{}{
							"name":      "test_tool",
							"arguments": map[string]interface{}{"x": 1},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	p := NewProvider(ts.URL, "llama3.1")
	content, calls, err := p.Chat(context.Background(), []llm.Message{{Role: llm.RoleUser, Content: "hi"}}, []llm.Tool{{Name: "test_tool"}})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "hi there" {
		t.Errorf("content = %q", content)
	}
	if len(calls) != 1 || calls[0].Name != "test_tool" {
		t.Errorf("calls = %v", calls)
	}
}

func TestProviderChatNoToolCalls(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"message": map[string]interface{}{
				"role":      "assistant",
				"content":   "plain",
				"tool_calls": []map[string]interface{}{},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	p := NewProvider(ts.URL, "llama3.1")
	content, calls, err := p.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "plain" {
		t.Errorf("content = %q", content)
	}
	if len(calls) != 0 {
		t.Errorf("calls = %v", calls)
	}
}

func TestProviderChatErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer ts.Close()

	p := NewProvider(ts.URL, "llama3.1")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestProviderChatInvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	p := NewProvider(ts.URL, "llama3.1")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestProviderChatNetworkError(t *testing.T) {
	p := NewProvider("http://127.0.0.1:1", "llama3.1")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("read error") }
func (errReader) Close() error             { return nil }

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestProviderChatReadError(t *testing.T) {
	p := NewProvider("http://localhost:11434", "llama3.1")
	p.client = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: errReader{}}, nil
	})}
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestProviderChatMarshalError(t *testing.T) {
	orig := marshalFn
	marshalFn = func(v any) ([]byte, error) { return nil, fmt.Errorf("marshal error") }
	defer func() { marshalFn = orig }()

	p := NewProvider("http://localhost:11434", "llama3.1")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestProviderChatRequestError(t *testing.T) {
	p := NewProvider("http://\x00invalid", "llama3.1")
	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Error("expected error")
	}
}
