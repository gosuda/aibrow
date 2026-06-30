// Package server wires the AIBrow REST API. It exposes a Gin engine with
// health checks, static frontend serving, and (in later phases) image
// processing endpoints. All handlers are annotated for Swaggo.
//
// @title       AIBrow API
// @version     1.0.0
// @description AI photo editor API with FFmpeg, LCH analysis, and LLM chat.
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/tools"
)

// Server wraps a gin.Engine and the application configuration.
type Server struct {
	Engine  *gin.Engine
	Config  Config
	Handler *Handler
}

// Config holds the runtime values needed by HTTP handlers.
type Config struct {
	UploadDir string
	OutputDir string
	Provider  llm.Provider
}

// New creates a Server with routes registered.
func New(cfg Config) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	s := &Server{Engine: r, Config: cfg}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.Engine.GET("/health", healthHandler)
	s.Engine.Static("/web", "web")
	s.Engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/web/index.html")
	})
	s.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	s.Handler = &Handler{
		Store:     NewStore(),
		Client:    ffmpeg.NewClient(),
		UploadDir: s.Config.UploadDir,
		OutputDir: s.Config.OutputDir,
		Provider:  s.Config.Provider,
	}
	s.Handler.Registry = tools.NewDefaultRegistry(s.Handler.Store, s.Handler.Client, s.Handler.OutputDir)
	api := s.Engine.Group("/api/v1")
	{
		api.POST("/images/upload", s.Handler.UploadImage)
		api.GET("/images/:id", s.Handler.GetImage)
		api.POST("/images/:id/analyze", s.Handler.AnalyzeImage)
		api.POST("/images/:id/adjust", s.Handler.AdjustImage)
		api.POST("/images/:id/convert", s.Handler.ConvertImage)
		api.GET("/images/:id/processed", s.Handler.GetProcessed)
		api.POST("/images/:id/export", s.Handler.ExportImage)
		api.POST("/chat", s.Handler.Chat)
		api.POST("/request", s.Handler.Request)
	}
}

// healthResponse is the JSON body returned by the health endpoint.
type healthResponse struct {
	Status string `json:"status"`
}

// healthHandler godoc
// @Summary      Health check
// @Description  Returns the current service health status.
// @Tags         system
// @Produce      json
// @Success      200  {object}  healthResponse
// @Router       /health [get]
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}
