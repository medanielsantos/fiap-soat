package httpserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

type Server struct {
	address string
	echo    *echo.Echo
}

func New(cfg config.Config, customerHandler *customerhttp.Handler) *Server {
	e := echo.New()
	e.Use(middleware.Recover())
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.Use(middleware.BodyLimit(1 << 20))
	customerHandler.RegisterRoutes(e.Group("/customers"))

	return &Server{address: cfg.HTTPAddress, echo: e}
}

func (s *Server) Run(ctx context.Context) error {
	routes := s.echo.Router().Routes()
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	log.Println("Registered HTTP routes:")
	for _, route := range routes {
		log.Printf("  %-7s %s", route.Method, route.Path)
	}

	err := (echo.StartConfig{
		Address:         s.address,
		HideBanner:      true,
		GracefulTimeout: 10 * time.Second,
	}).Start(ctx, s.echo)
	if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
