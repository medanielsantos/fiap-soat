package httpserver

import (
	"net/http"

	"github.com/labstack/echo/v5"
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	vehiclehttp "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/in/http"
)

type Router struct {
	engine          *echo.Echo
	customerHandler *customerhttp.Handler
	vehicleHandler  *vehiclehttp.Handler
}

func NewRouter(engine *echo.Echo, customerHandler *customerhttp.Handler, vehicleHandler *vehiclehttp.Handler) *Router {
	return &Router{
		engine:          engine,
		customerHandler: customerHandler,
		vehicleHandler:  vehicleHandler,
	}
}

func (router *Router) Register() {
	router.engine.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := router.engine.Group("/api/v1")
	router.customerHandler.RegisterRoutes(v1.Group("/customers"))
	router.vehicleHandler.RegisterRoutes(v1.Group("/vehicles"))
}
