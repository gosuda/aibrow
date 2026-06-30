package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(Tool{Name: "test", Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
		return "ok", nil
	}})

	tool, ok := r.Get("test")
	if !ok {
		t.Fatal("expected tool")
	}
	if tool.Name != "test" {
		t.Errorf("name = %q", tool.Name)
	}
}

func TestRegistryDuplicatePanics(t *testing.T) {
	r := NewRegistry()
	r.Register(Tool{Name: "dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate registration")
		}
	}()
	r.Register(Tool{Name: "dup"})
}

func TestRegistryExecute(t *testing.T) {
	r := NewRegistry()
	r.Register(Tool{Name: "echo", Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
		return string(input), nil
	}})

	out, err := r.Execute(context.Background(), "echo", json.RawMessage(`"hi"`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != `"hi"` {
		t.Errorf("out = %v", out)
	}
}

func TestRegistryExecuteMissing(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "missing", nil)
	if err == nil {
		t.Error("expected error for missing tool")
	}
}

func TestRegistryNames(t *testing.T) {
	r := NewRegistry()
	r.Register(Tool{Name: "a"})
	r.Register(Tool{Name: "b"})
	names := r.Names()
	if len(names) != 2 {
		t.Errorf("names = %v", names)
	}
}
