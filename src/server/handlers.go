package server

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/tools"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	Store      *Store
	Client     *ffmpeg.Client
	UploadDir  string
	OutputDir  string
	Provider   llm.Provider
	Registry   *tools.Registry
}

// uploadResponse is returned by the image upload endpoint.
type uploadResponse struct {
	ID string `json:"id"`
}

// UploadImage godoc
// @Summary      Upload an image
// @Description  Uploads an image file for editing.
// @Tags         images
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "image file"
// @Success      200    {object}  uploadResponse
// @Failure      400    {object}  map[string]string
// @Router       /api/v1/images/upload [post]
func (h *Handler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing image"})
		return
	}
	ext := filepath.Ext(file.Filename)
	id := uuid.New().String()
	path := filepath.Join(h.UploadDir, id+ext)
	if err := c.SaveUploadedFile(file, path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	record := h.Store.Save(ImageRecord{OriginalPath: path})
	c.JSON(http.StatusOK, uploadResponse{ID: record.ID})
}

// GetImage godoc
// @Summary      Get an image
// @Description  Serves the original image by ID.
// @Tags         images
// @Produce      octet-stream
// @Param        id   path      string  true  "image id"
// @Success      200  {file}    binary
// @Failure      404  {object}  map[string]string
// @Router       /api/v1/images/{id} [get]
func (h *Handler) GetImage(c *gin.Context) {
	id := c.Param("id")
	record, ok := h.Store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.File(record.OriginalPath)
}

// AnalyzeImage godoc
// @Summary      Analyze an image
// @Description  Runs LCH analysis on the uploaded image.
// @Tags         images
// @Produce      json
// @Param        id   path      string  true  "image id"
// @Success      200  {object}  ffmpeg.Analysis
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /api/v1/images/{id}/analyze [post]
func (h *Handler) AnalyzeImage(c *gin.Context) {
	id := c.Param("id")
	record, ok := h.Store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	analysis, err := h.Client.Analyze(record.OriginalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	record.Analysis = analysis
	_ = h.Store.Update(record)
	c.JSON(http.StatusOK, analysis)
}

// AdjustImage godoc
// @Summary      Adjust an image
// @Description  Applies adjustment parameters to produce the AI edit.
// @Tags         images
// @Accept       json
// @Produce      octet-stream
// @Param        id      path      string             true  "image id"
// @Param        params  body      ffmpeg.AdjustParams  true  "adjustment params"
// @Success      200     {file}    binary
// @Failure      400     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /api/v1/images/{id}/adjust [post]
func (h *Handler) AdjustImage(c *gin.Context) {
	id := c.Param("id")
	record, ok := h.Store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var params ffmpeg.AdjustParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	output := filepath.Join(h.OutputDir, id+"_edited.png")
	if err := h.Client.Adjust(record.OriginalPath, output, params); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	record.ProcessedPath = output
	record.Params = params
	_ = h.Store.Update(record)
	c.File(output)
}

// ConvertImage godoc
// @Summary      Convert image format
// @Description  Saves the processed image in a different format.
// @Tags         images
// @Produce      octet-stream
// @Param        id      path      string  true  "image id"
// @Param        format  query     string  true  "target format"
// @Param        quality query     string  false "quality"
// @Success      200     {file}    binary
// @Failure      400     {object}  map[string]string
// @Failure      404     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /api/v1/images/{id}/convert [post]
func (h *Handler) ConvertImage(c *gin.Context) {
	id := c.Param("id")
	record, ok := h.Store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	format := c.Query("format")
	if format == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing format"})
		return
	}
	source := record.OriginalPath
	if record.ProcessedPath != "" {
		source = record.ProcessedPath
	}
	output := filepath.Join(h.OutputDir, fmt.Sprintf("%s_converted.%s", id, format))
	if err := h.Client.Convert(source, output, map[string]string{"format": format, "quality": c.Query("quality")}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.File(output)
}
