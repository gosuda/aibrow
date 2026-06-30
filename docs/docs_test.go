package docs

import (
	"testing"
)

func TestSwaggerInfoInitialized(t *testing.T) {
	if SwaggerInfo.Title != "AIBrow API" {
		t.Errorf("title = %q, want AIBrow API", SwaggerInfo.Title)
	}
	if SwaggerInfo.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", SwaggerInfo.Version)
	}
}
