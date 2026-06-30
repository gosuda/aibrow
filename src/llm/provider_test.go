package llm

import (
	"testing"
)

func TestToolResultMessage(t *testing.T) {
	result := ToolResult{Name: "analyze_image", Content: "bright"}
	msg := ToolResultMessage(result)
	if msg.Role != RoleUser {
		t.Errorf("role = %q, want user", msg.Role)
	}
	want := "Tool result for analyze_image: bright"
	if msg.Content != want {
		t.Errorf("content = %q, want %q", msg.Content, want)
	}
}
