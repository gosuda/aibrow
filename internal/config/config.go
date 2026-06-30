// Package config loads application settings from environment variables
// and an optional .env file. It also exposes port selection helpers.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// AppConfig holds all runtime configuration values.
type AppConfig struct {
	RestPort        int
	SwaggerPort     int
	LogPath         string
	UploadDir       string
	OutputDir       string
	OpenAIAPIKey    string
	GeminiAPIKey    string
	OllamaBaseURL   string
	DefaultProvider string
}

// Load reads .env (if present) and environment variables into AppConfig.
func Load(path string) (*AppConfig, error) {
	if path == "" {
		path = "configs/.env"
	}
	// .env is optional; ignore ErrMissingFile.
	_ = godotenv.Load(path)

	cfg := &AppConfig{
		RestPort:        intEnv("REST_PORT", 8080),
		SwaggerPort:     intEnv("SWAGGER_PORT", 8081),
		LogPath:         stringEnv("LOG_PATH", "logs/error.log"),
		UploadDir:       stringEnv("UPLOAD_DIR", "uploads"),
		OutputDir:       stringEnv("OUTPUT_DIR", "outputs"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		GeminiAPIKey:    os.Getenv("GEMINI_API_KEY"),
		OllamaBaseURL:   stringEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		DefaultProvider: stringEnv("DEFAULT_PROVIDER", "openai"),
	}

	if err := cfg.ensureDirs(); err != nil {
		return nil, fmt.Errorf("config: ensure dirs: %w", err)
	}

	return cfg, nil
}

func (c *AppConfig) ensureDirs() error {
	for _, dir := range []string{c.UploadDir, c.OutputDir, "logs"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func intEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func stringEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
