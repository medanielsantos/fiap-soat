package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

type Server struct {
	address string
	echo    *echo.Echo
}

func New(cfg config.Config) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return &Server{address: cfg.HTTPAddress, echo: e}
}

func (s *Server) Run(ctx context.Context) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- s.echo.Start(s.address)
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.echo.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}
