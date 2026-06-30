package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yjlee/aibrow/src/llm"
)

type requestProvider struct {
	resp  string
	calls []llm.ToolCall
	err   error
	n     int
}

func (r *requestProvider) Chat(ctx context.Context, messages []llm.Message, tools []llm.Tool) (string, []llm.ToolCall, error) {
	r.n++
	if r.n > 1 {
		return "done", nil, r.err
	}
	return r.resp, r.calls, r.err
}

func buildRequestMultipart(t *testing.T, imagePath, prompt string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if imagePath != "" {
		file, err := os.Open(imagePath)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer file.Close()
		part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			t.Fatalf("copy: %v", err)
		}
	}
	if prompt != "" {
		_ = writer.WriteField("prompt", prompt)
	}
	writer.Close()
	return &body, writer.FormDataContentType()
}

func TestRequestProviderNotConfigured(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)

	body, ct := buildRequestMultipart(t, img, "make it brighter")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestRequestRegistryNotConfigured(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	srv.Handler.Provider = &requestProvider{}
	srv.Handler.Registry = nil

	body, ct := buildRequestMultipart(t, img, "make it brighter")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestRequestMissingImage(t *testing.T) {
	srv, _ := setupTestServer(t)
	srv.Handler.Provider = &requestProvider{}

	body, ct := buildRequestMultipart(t, "", "make it brighter")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRequestMissingPrompt(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	srv.Handler.Provider = &requestProvider{}

	body, ct := buildRequestMultipart(t, img, "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRequestSaveError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	srv.Handler.Provider = &requestProvider{}
	// Make upload dir a file so SaveUploadedFile fails.
	srv.Handler.UploadDir = filepath.Join(dir, "uploads_file")
	_ = os.WriteFile(srv.Handler.UploadDir, []byte("x"), 0o644)

	body, ct := buildRequestMultipart(t, img, "make it brighter")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestRequestProviderError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	srv.Handler.Provider = &requestProvider{err: errors.New("model failed")}

	body, ct := buildRequestMultipart(t, img, "make it brighter")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestRequestSuccess(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	srv.Handler.Provider = &requestProvider{resp: "Adjusted the image"}

	body, ct := buildRequestMultipart(t, img, "make it brighter")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp requestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ImageID == "" {
		t.Error("expected image id")
	}
	if resp.Message != "Adjusted the image" {
		t.Errorf("message = %q", resp.Message)
	}
}

func TestRequestWithToolCall(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	srv.Handler.Provider = &requestProvider{
		calls: []llm.ToolCall{{Name: "analyze_image", Arguments: json.RawMessage(`{"image_id":"img-1"}`)}},
	}

	body, ct := buildRequestMultipart(t, img, "analyze")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/request", body)
	req.Header.Set("Content-Type", ct)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp requestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("tool calls = %v", resp.ToolCalls)
	}
}

func TestExportImage(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/export", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestExportImageRegistryNotConfigured(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)
	srv.Handler.Registry = nil

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/export", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestExportImageNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/missing/export", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestExportImageWriteError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)
	// Make outputDir a file so Excel save fails.
	srv.Handler.OutputDir = filepath.Join(dir, "output_file")
	_ = os.WriteFile(srv.Handler.OutputDir, []byte("x"), 0o644)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/export", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestGetProcessed(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	record, _ := srv.Handler.Store.Get(id)
	record.ProcessedPath = filepath.Join(dir, "processed.png")
	_ = os.WriteFile(record.ProcessedPath, []byte("fake"), 0o644)
	_ = srv.Handler.Store.Update(record)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+id+"/processed", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGetProcessedNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/missing/processed", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGetProcessedMissingPath(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+id+"/processed", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

var _ = strings.NewReader
