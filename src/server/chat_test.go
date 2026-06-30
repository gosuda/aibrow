package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/tools"
)

type stubProvider struct {
	resp      string
	call      []llm.ToolCall
	err       error
	callCount int
}

func (s *stubProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.Tool) (string, []llm.ToolCall, error) {
	s.callCount++
	if s.callCount > 1 {
		return "done", nil, s.err
	}
	return s.resp, s.call, s.err
}

func TestChatProviderNotConfigured(t *testing.T) {
	srv, _ := setupTestServer(t)
	body := `{"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestChatRegistryNotConfigured(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &stubProvider{resp: "hello"}
	srv.Handler.Registry = nil

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestChatBadJSON(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &stubProvider{}
	srv.Handler.Registry = tools.NewRegistry()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestChatMissingMessages(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &stubProvider{}
	srv.Handler.Registry = tools.NewRegistry()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestChatProviderError(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &stubProvider{err: errors.New("model failed")}
	srv.Handler.Registry = tools.NewRegistry()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestChatSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &stubProvider{resp: "hello"}
	srv.Handler.Registry = tools.NewRegistry()

	body := `{"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Message != "hello" {
		t.Errorf("message = %q", resp.Message)
	}
}

func TestChatWithToolCall(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &stubProvider{
		call: []llm.ToolCall{{Name: "echo", Arguments: json.RawMessage(`{"x":1}`)}},
	}
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name:        "echo",
		Description: "echo",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			return "ok", nil
		},
	})
	srv.Handler.Registry = registry

	body := `{"messages":[{"role":"user","content":"go"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}
