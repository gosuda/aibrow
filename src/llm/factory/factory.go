// Package factory wires LLM providers from environment variables.
package factory

import (
	"context"
	"fmt"
	"os"

	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/llm/gemini"
	"github.com/yjlee/aibrow/src/llm/ollama"
	"github.com/yjlee/aibrow/src/llm/openai"
)

// NewProviderFromEnv picks the first available provider based on environment
// variables. Priority: OpenAI, Gemini, Ollama.
func NewProviderFromEnv(ctx context.Context) (llm.Provider, error) {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		baseURL := os.Getenv("OPENAI_BASE_URL")
		model := os.Getenv("OPENAI_MODEL")
		return openai.NewProvider(key, baseURL, model), nil
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		model := os.Getenv("GEMINI_MODEL")
		return gemini.NewProvider(ctx, key, model)
	}
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	return ollama.NewProvider(baseURL, model), nil
}

// NewProvider creates a named provider from explicit credentials.
func NewProvider(ctx context.Context, name string) (llm.Provider, error) {
	switch name {
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return openai.NewProvider(key, os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_MODEL")), nil
	case "gemini":
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY not set")
		}
		return gemini.NewProvider(ctx, key, os.Getenv("GEMINI_MODEL"))
	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return ollama.NewProvider(baseURL, os.Getenv("OLLAMA_MODEL")), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
