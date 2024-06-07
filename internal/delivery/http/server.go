package http

import (
	"blum-test/common/config"
	"blum-test/common/logger"
	"blum-test/internal/service"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	mlogger "github.com/gofiber/fiber/v2/middleware/logger"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	. "blum-test/docs"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.AppConfig
	app *fiber.App
	svc *service.RateCalculator
}

func NewServer(cfg *config.AppConfig, svc *service.RateCalculator) *Server {
	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		JSONEncoder:   json.Marshal,
	})
	app.Use(mlogger.New())

	return &Server{
		cfg: cfg,
		app: app,
		svc: svc,
	}
}

func (s *Server) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel

	host := fmt.Sprintf("%s:%d", s.cfg.HTTPServer.Host, s.cfg.HTTPServer.Port)
	SwaggerInfo.Host = host
	SwaggerInfo.BasePath = "/v0"
	s.app.Get("/swagger/*", fiberSwagger.FiberWrapHandler())

	api := s.app.Group("/v0")
	api.Get("/convert", s.Convert)

	go func() {
		if err := s.app.Listen(host); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.JSONLogger.Error("error while serving", slog.Any("error", err))
		}
	}()

	<-ctx.Done()

	logger.JSONLogger.Info("shutting down server...")
	if err := s.app.ShutdownWithTimeout(s.cfg.HTTPServer.ShutdownTimeout); err != nil {
		return fmt.Errorf("error while shutdowning http server: %w", err)
	}
	return nil
}

func (s *Server) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
