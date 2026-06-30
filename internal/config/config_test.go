package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.RestPort != 8080 {
		t.Errorf("RestPort = %d, want 8080", cfg.RestPort)
	}
	if cfg.SwaggerPort != 8081 {
		t.Errorf("SwaggerPort = %d, want 8081", cfg.SwaggerPort)
	}
	if cfg.LogPath != "logs/error.log" {
		t.Errorf("LogPath = %q, want logs/error.log", cfg.LogPath)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("REST_PORT", "9090")
	t.Setenv("SWAGGER_PORT", "9091")
	t.Setenv("LOG_PATH", "logs/test.log")
	t.Setenv("OPENAI_API_KEY", "sk-test")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.RestPort != 9090 {
		t.Errorf("RestPort = %d, want 9090", cfg.RestPort)
	}
	if cfg.SwaggerPort != 9091 {
		t.Errorf("SwaggerPort = %d, want 9091", cfg.SwaggerPort)
	}
	if cfg.LogPath != "logs/test.log" {
		t.Errorf("LogPath = %q, want logs/test.log", cfg.LogPath)
	}
	if cfg.OpenAIAPIKey != "sk-test" {
		t.Errorf("OpenAIAPIKey = %q, want sk-test", cfg.OpenAIAPIKey)
	}
}

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("REST_PORT=7070\nSWAGGER_PORT=7071\n"), 0o644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.RestPort != 7070 {
		t.Errorf("RestPort = %d, want 7070", cfg.RestPort)
	}
	if cfg.SwaggerPort != 7071 {
		t.Errorf("SwaggerPort = %d, want 7071", cfg.SwaggerPort)
	}
}

func TestIntEnvInvalidFallsBack(t *testing.T) {
	t.Setenv("REST_PORT", "not-a-number")
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.RestPort != 8080 {
		t.Errorf("RestPort = %d, want 8080", cfg.RestPort)
	}
}

func TestLoadEnsureDirsError(t *testing.T) {
	// Use a path that cannot be created as a directory to trigger ensureDirs error.
	t.Setenv("UPLOAD_DIR", "/dev/null/upload")
	_, err := Load("")
	if err == nil {
		t.Error("expected error when ensureDirs fails")
	}
}

func TestEnsureDirsPropagatesError(t *testing.T) {
	cfg := &AppConfig{
		UploadDir: "/proc/invalid-upload",
		OutputDir: "outputs",
	}
	err := cfg.ensureDirs()
	if err == nil {
		t.Error("expected ensureDirs to fail for invalid path")
	}
}
