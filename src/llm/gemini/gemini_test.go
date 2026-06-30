package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/yjlee/aibrow/src/llm"
	"google.golang.org/api/option"
)

func setupTestProvider(t *testing.T, handler http.HandlerFunc) (*Provider, *httptest.Server) {
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	orig := newGenaiClient
	newGenaiClient = func(ctx context.Context, opts ...option.ClientOption) (*genai.Client, error) {
		return genai.NewClient(ctx, append(opts, option.WithEndpoint(ts.URL))...)
	}
	t.Cleanup(func() { newGenaiClient = orig })

	p, err := NewProvider(context.Background(), "test-key", "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	return p, ts
}

func rawStream(content string) string {
	return `[{"candidates":[{"content":{"role":"model","parts":` + content + `}}]}]`
}

func TestNewProviderDefaults(t *testing.T) {
	p, err := NewProvider(context.Background(), "test-key", "")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if p.model != "gemini-1.5-flash" {
		t.Errorf("model = %q, want gemini-1.5-flash", p.model)
	}
}

func TestNewProviderCustomModel(t *testing.T) {
	p, err := NewProvider(context.Background(), "test-key", "gemini-pro")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if p.model != "gemini-pro" {
		t.Errorf("model = %q", p.model)
	}
}

func TestNewProviderError(t *testing.T) {
	orig := newGenaiClient
	newGenaiClient = func(ctx context.Context, opts ...option.ClientOption) (*genai.Client, error) {
		return nil, fmt.Errorf("boom")
	}
	defer func() { newGenaiClient = orig }()

	_, err := NewProvider(context.Background(), "test-key", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderChatTextAndToolCall(t *testing.T) {
	p, _ := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		parts := `[{"text":"hello"},{"functionCall":{"name":"analyze_image","args":{"image_id":"img-1"}}}]`
		w.Write([]byte(rawStream(parts)))
	})

	schema := json.RawMessage(`{"type":"object","properties":{"image_id":{"type":"string"},"tags":{"type":"array","items":{"type":"integer"}},"unknown":{"type":"unsupported"}},"required":["image_id"]}`)
	content, calls, err := p.Chat(context.Background(),
		[]llm.Message{{Role: llm.RoleUser, Content: "analyze"}},
		[]llm.Tool{
			{Name: "analyze_image", Description: "analyze", Parameters: schema},
			{Name: "noop"},
		})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "hello" {
		t.Errorf("content = %q", content)
	}
	if len(calls) != 1 || calls[0].Name != "analyze_image" {
		t.Errorf("calls = %v", calls)
	}
}

func TestProviderChatNoTools(t *testing.T) {
	p, _ := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(rawStream(`[{"text":"ok"}]`)))
	})

	content, calls, err := p.Chat(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if content != "ok" {
		t.Errorf("content = %q", content)
	}
	if len(calls) != 0 {
		t.Errorf("calls = %v", calls)
	}
}

func TestProviderChatSendError(t *testing.T) {
	p, _ := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})

	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderChatNoCandidates(t *testing.T) {
	p, _ := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal([]*genai.GenerateContentResponse{{Candidates: []*genai.Candidate{}}})
		w.Write(resp)
	})

	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderChatNilContent(t *testing.T) {
	p, _ := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal([]*genai.GenerateContentResponse{{Candidates: []*genai.Candidate{{Content: nil}}}})
		w.Write(resp)
	})

	_, _, err := p.Chat(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderChatInvalidSchemaIgnored(t *testing.T) {
	p, _ := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(rawStream(`[{"text":"done"}]`)))
	})

	_, _, err := p.Chat(context.Background(), nil, []llm.Tool{
		{Name: "bad", Parameters: json.RawMessage(`{invalid`)},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
}

func TestSchemaFromJSON(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"count":{"type":"number"},"flag":{"type":"boolean"},"items":{"type":"array","items":{"type":"string"}}}}`)
	out, err := schemaFromJSON(schema)
	if err != nil {
		t.Fatalf("schemaFromJSON: %v", err)
	}
	if out.Type != genai.TypeObject {
		t.Errorf("type = %v", out.Type)
	}
	if out.Properties["count"].Type != genai.TypeNumber {
		t.Errorf("count type = %v", out.Properties["count"].Type)
	}
	if out.Properties["flag"].Type != genai.TypeBoolean {
		t.Errorf("flag type = %v", out.Properties["flag"].Type)
	}
	if out.Properties["items"].Items.Type != genai.TypeString {
		t.Errorf("items item type = %v", out.Properties["items"].Items.Type)
	}
}

func TestSchemaFromJSONPropertyError(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"bad":{"type":123}}}`)
	_, err := schemaFromJSON(schema)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSchemaFromJSONItemsError(t *testing.T) {
	schema := json.RawMessage(`{"type":"array","items":{"type":123}}`)
	_, err := schemaFromJSON(schema)
	if err == nil {
		t.Fatal("expected error")
	}
}
