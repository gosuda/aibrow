package factory

import (
	"context"
	"testing"
)

func TestNewProviderFromEnvOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_BASE_URL", "http://openai.example.com")
	t.Setenv("OPENAI_MODEL", "gpt-4o-mini")
	p, err := NewProviderFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderFromEnvGemini(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "gemini-pro")
	p, err := NewProviderFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderFromEnvOllamaExplicitURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")
	t.Setenv("OLLAMA_MODEL", "llama3.1")
	p, err := NewProviderFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderFromEnvOllamaDefaultURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_MODEL", "llama3.1")
	p, err := NewProviderFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewProviderFromEnv: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderNamedOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_BASE_URL", "http://openai.example.com")
	t.Setenv("OPENAI_MODEL", "gpt-4o-mini")
	p, err := NewProvider(context.Background(), "openai")
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderNamedGemini(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("GEMINI_MODEL", "gemini-pro")
	p, err := NewProvider(context.Background(), "gemini")
	if err != nil {
		t.Fatalf("NewProvider gemini: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderNamedOllama(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://localhost:11434")
	t.Setenv("OLLAMA_MODEL", "llama3.1")
	p, err := NewProvider(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("NewProvider ollama: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderNamedOllamaDefaultURL(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_MODEL", "llama3.1")
	p, err := NewProvider(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("NewProvider ollama default: %v", err)
	}
	if p == nil {
		t.Error("expected provider")
	}
}

func TestNewProviderNamedErrors(t *testing.T) {
	_, err := NewProvider(context.Background(), "unknown")
	if err == nil {
		t.Error("expected error for unknown provider")
	}

	t.Setenv("OPENAI_API_KEY", "")
	_, err = NewProvider(context.Background(), "openai")
	if err == nil {
		t.Error("expected error when openai key missing")
	}

	t.Setenv("GEMINI_API_KEY", "")
	_, err = NewProvider(context.Background(), "gemini")
	if err == nil {
		t.Error("expected error when gemini key missing")
	}
}
