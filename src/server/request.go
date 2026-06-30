package server

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yjlee/aibrow/src/chat"
	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/store"
	"github.com/yjlee/aibrow/src/tools"
)

// requestResponse is returned by the natural-language request endpoint.
type requestResponse struct {
	ImageID   string                `json:"image_id"`
	Message   string                `json:"message"`
	ToolCalls []llm.ToolCall        `json:"tool_calls,omitempty"`
	Results   []chat.ToolCallResult `json:"results,omitempty"`
}

// requestResponseSwagger is a swagger-friendly mirror of requestResponse.
type requestResponseSwagger struct {
	ImageID   string                   `json:"image_id"`
	Message   string                   `json:"message"`
	ToolCalls []chatResponseToolCall   `json:"tool_calls,omitempty"`
	Results   []chatResponseToolResult `json:"results,omitempty"`
}

// Request godoc
// @Summary      Natural language photo edit request
// @Description  Uploads an image and lets the configured LLM orchestrate analysis, adjustment, conversion, and export tools.
// @Tags         request
// @Accept       multipart/form-data
// @Produce      json
// @Param        image   formData  file    true  "image file"
// @Param        prompt  formData  string  true  "natural language instruction"
// @Success      200     {object}  requestResponseSwagger
// @Failure      400     {object}  map[string]string
// @Failure      503     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /api/v1/request [post]
func (h *Handler) Request(c *gin.Context) {
	if h.Provider == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LLM provider not configured"})
		return
	}
	if h.Registry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tool registry not configured"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing image"})
		return
	}
	prompt := c.PostForm("prompt")
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing prompt"})
		return
	}

	ext := filepath.Ext(file.Filename)
	uploadID := uuid.New().String()
	path := filepath.Join(h.UploadDir, uploadID+ext)
	if err := c.SaveUploadedFile(file, path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	record := h.Store.Save(store.ImageRecord{OriginalPath: path})

	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are an AI photo editor. Use the available tools to fulfill the user's request. Call analyze_image first, then adjust_image if edits are needed. Use convert_format or export_excel only when explicitly requested."},
		{Role: llm.RoleUser, Content: prompt},
	}

	chatResp, err := chat.Run(c.Request.Context(), h.Provider, h.Registry, messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	record, _ = h.Store.Get(record.ID)
	resp := requestResponse{
		ImageID:   record.ID,
		Message:   chatResp.Message,
		ToolCalls: chatResp.ToolCalls,
		Results:   chatResp.Results,
	}
	c.JSON(http.StatusOK, resp)
}

// ExportImage godoc
// @Summary      Export image preset to Excel
// @Description  Exports the analysis and adjustment parameters for an image as an .xlsx file.
// @Tags         images
// @Produce      octet-stream
// @Param        id   path      string  true  "image id"
// @Success      200  {file}    binary
// @Failure      404  {object}  map[string]string
// @Failure      503  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/images/{id}/export [post]
func (h *Handler) ExportImage(c *gin.Context) {
	if h.Registry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tool registry not configured"})
		return
	}

	id := c.Param("id")
	record, ok := h.Store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	output := filepath.Join(h.OutputDir, record.ID+"_preset.xlsx")
	if err := tools.WritePresetExcel(output, record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.File(output)
}

// GetProcessed godoc
// @Summary      Get processed image
// @Description  Serves the edited/processed image by ID if one exists.
// @Tags         images
// @Produce      octet-stream
// @Param        id   path      string  true  "image id"
// @Success      200  {file}    binary
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/images/{id}/processed [get]
func (h *Handler) GetProcessed(c *gin.Context) {
	id := c.Param("id")
	record, ok := h.Store.Get(id)
	if !ok || record.ProcessedPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.File(record.ProcessedPath)
}


