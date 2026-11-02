package httpserver

import (
	"net/http"

	"github.com/labstack/echo/v5"
	customerhttp "github.com/medanielsantos/fiap-soat/internal/customer/adapters/in/http"
	identityhttp "github.com/medanielsantos/fiap-soat/internal/identity/adapters/in/http"
	identityapp "github.com/medanielsantos/fiap-soat/internal/identity/application"
	cataloghttp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/in/http"
	vehiclehttp "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/in/http"
	workorderhttp "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/in/http"
)

type Router struct {
	engine           *echo.Echo
	customerHandler  *customerhttp.Handler
	vehicleHandler   *vehiclehttp.Handler
	catalogHandler   *cataloghttp.Handler
	workOrderHandler *workorderhttp.Handler
	identityHandler  *identityhttp.Handler
}

func NewRouter(engine *echo.Echo, customerHandler *customerhttp.Handler, vehicleHandler *vehiclehttp.Handler, catalogHandler *cataloghttp.Handler, workOrderHandler *workorderhttp.Handler, identityHandler *identityhttp.Handler) *Router {
	return &Router{
		engine:           engine,
		customerHandler:  customerHandler,
		vehicleHandler:   vehicleHandler,
		catalogHandler:   catalogHandler,
		workOrderHandler: workOrderHandler,
		identityHandler:  identityHandler,
	}
}

func (router *Router) Register() {
	router.engine.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := router.engine.Group("/api/v1")
	router.identityHandler.RegisterRoutes(v1)
	router.customerHandler.RegisterClientRoutes(v1)
	admin := v1.Group("")
	admin.Use(router.identityHandler.RequireAuth())
	router.customerHandler.RegisterRoutes(admin.Group("/customers"))
	router.vehicleHandler.RegisterRoutes(admin.Group("/vehicles"))
	router.catalogHandler.RegisterRoutes(admin, router.identityHandler.RequirePermission(identityapp.PermissionProductsCreate), router.identityHandler.RequirePermission(identityapp.PermissionProductsDelete))
	router.workOrderHandler.RegisterRoutes(admin.Group("/work-orders"))
}
