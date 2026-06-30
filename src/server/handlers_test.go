package server

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yjlee/aibrow/src/ffmpeg"
	ff "github.com/u2takey/ffmpeg-go"
)

func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := Config{UploadDir: filepath.Join(dir, "uploads"), OutputDir: filepath.Join(dir, "outputs")}
	_ = os.MkdirAll(cfg.UploadDir, 0o755)
	_ = os.MkdirAll(cfg.OutputDir, 0o755)
	return New(cfg), dir
}

func createTestImage(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=red:s=64x64", "-frames:v", "1", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available: %v\n%s", err, out)
	}
}

func uploadImage(t *testing.T, srv *Server, path string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()
	part, err := writer.CreateFormFile("image", filepath.Base(path))
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		t.Fatalf("copy: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("upload status = %d, body = %s", w.Code, w.Body.String())
	}
	return extractID(w.Body.String())
}

func extractID(body string) string {
	// body is {"id":"..."}
	body = strings.TrimPrefix(body, `{"id":"`)
	body = strings.TrimSuffix(body, `"}`)
	return body
}

func TestUploadImage(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)
	if id == "" {
		t.Error("expected id")
	}
	if srv.Handler.Store.Len() != 1 {
		t.Errorf("store len = %d, want 1", srv.Handler.Store.Len())
	}
}

func TestGetImage(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/"+id, nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGetImageNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/missing", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAnalyzeImage(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/analyze", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "width") {
		t.Error("expected width in response")
	}
}

func TestAdjustImage(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	body := `{"brightness":0.1,"contrast":1.1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestConvertImage(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/convert?format=jpg&quality=5", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestConvertImageMissingFormat(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/convert", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestUploadImageMissing(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/upload", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestAnalyzeImageNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/missing/analyze", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAdjustImageNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/missing/adjust", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAdjustImageBadJSON(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/adjust", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestConvertImageNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/missing/convert?format=png", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestUploadImageSaveError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)

	// Make upload dir a file so SaveUploadedFile fails.
	srv.Handler.UploadDir = filepath.Join(dir, "uploads_file")
	_ = os.WriteFile(srv.Handler.UploadDir, []byte("x"), 0o644)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, _ := os.Open(img)
	part, _ := writer.CreateFormFile("image", filepath.Base(img))
	_, _ = io.Copy(part, file)
	file.Close()
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestAnalyzeImageClientError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	srv.Handler.Client = ffmpeg.NewClient(ffmpeg.WithExecutor(&fakeProbeExecutor{err: errors.New("probe failed")}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/analyze", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestAdjustImageClientError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	orig := ffmpeg.RunFFmpegFn
	ffmpeg.RunFFmpegFn = func(binary string, args ...string) error { return errors.New("adjust failed") }
	defer func() { ffmpeg.RunFFmpegFn = orig }()

	body := `{"brightness":0.1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/adjust", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestConvertImageClientError(t *testing.T) {
	srv, dir := setupTestServer(t)
	img := filepath.Join(dir, "test.png")
	createTestImage(t, img)
	id := uploadImage(t, srv, img)

	// Set a processed path so the source-selection branch is exercised.
	record, _ := srv.Handler.Store.Get(id)
	record.ProcessedPath = filepath.Join(dir, "processed.png")
	_ = srv.Handler.Store.Update(record)

	orig := ffmpeg.RunFFmpegFn
	ffmpeg.RunFFmpegFn = func(binary string, args ...string) error { return errors.New("convert failed") }
	defer func() { ffmpeg.RunFFmpegFn = orig }()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images/"+id+"/convert?format=png", nil)
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

type fakeProbeExecutor struct {
	err error
}

func (f *fakeProbeExecutor) Probe(fileName string, kwargs ...ff.KwArgs) (string, error) {
	return "", f.err
}

func (f *fakeProbeExecutor) Run(stream *ff.Stream, options ...ff.CompilationOption) error {
	return nil
}

// ensure gin context type is referenced.
var _ *gin.Context
