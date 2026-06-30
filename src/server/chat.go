package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yjlee/aibrow/src/chat"
	"github.com/yjlee/aibrow/src/llm"
)

// chatRequest is the body for POST /api/v1/chat.
type chatRequest struct {
	Messages []llm.Message `json:"messages" binding:"required"`
}

// chatResponseSwagger is a swagger-friendly mirror of chat.Response. It avoids
// referencing json.RawMessage, which swaggo cannot parse across packages.
type chatResponseSwagger struct {
	Message   string                   `json:"message"`
	ToolCalls []chatResponseToolCall   `json:"tool_calls,omitempty"`
	Results   []chatResponseToolResult `json:"results,omitempty"`
}

type chatResponseToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponseToolResult struct {
	Name   string      `json:"name"`
	Input  string      `json:"input"`
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Chat godoc
// @Summary      Chat with the AI photo editor
// @Description  Sends a conversation to the configured LLM and executes any requested tools.
// @Tags         chat
// @Accept       json
// @Produce      json
// @Param        request  body      chatRequest  true  "chat messages"
// @Success      200      {object}  chatResponseSwagger
// @Failure      400      {object}  map[string]string
// @Failure      503      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /api/v1/chat [post]
func (h *Handler) Chat(c *gin.Context) {
	if h.Provider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LLM provider not configured"})
		return
	}
	if h.Registry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tool registry not configured"})
		return
	}

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := chat.Run(c.Request.Context(), h.Provider, h.Registry, req.Messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
