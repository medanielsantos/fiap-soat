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
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
)

type Server struct {
	address string
	echo    *echo.Echo
}

func New(cfg config.Config, router *Router) *Server {
	router.engine.Use(middleware.Recover())
	router.engine.Use(middleware.BodyLimit(1 << 20))
	router.Register()

	return &Server{address: cfg.HTTPAddress, echo: router.engine}
}

func NewEngine() *echo.Echo { return echo.New() }

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
