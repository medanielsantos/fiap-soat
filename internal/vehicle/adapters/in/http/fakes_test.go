package vehiclehttp

import (
	"context"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

type handlerVehicleStore struct {
	vehicles map[string]domain.Vehicle
}

func (store *handlerVehicleStore) Save(_ context.Context, vehicle domain.Vehicle) error {
	for _, existing := range store.vehicles {
		if existing.Plate() == vehicle.Plate() {
			return application.ErrDuplicatePlate
		}
	}
	store.vehicles[vehicle.ID()] = vehicle
	return nil
}

func (store *handlerVehicleStore) FindByID(_ context.Context, id string) (domain.Vehicle, error) {
	vehicle, ok := store.vehicles[id]
	if !ok {
		return domain.Vehicle{}, application.ErrVehicleNotFound
	}
	return vehicle, nil
}

func (store *handlerVehicleStore) List(_ context.Context, filter application.ListFilter, params pagination.Params) (pagination.Page[domain.Vehicle], error) {
	vehicles := make([]domain.Vehicle, 0, len(store.vehicles))
	for _, vehicle := range store.vehicles {
		if filter.CustomerID != "" && vehicle.CustomerID() != filter.CustomerID {
			continue
		}
		vehicles = append(vehicles, vehicle)
	}
	return pagination.NewPage(vehicles, params, int64(len(vehicles))), nil
}

func (store *handlerVehicleStore) Update(_ context.Context, vehicle domain.Vehicle) error {
	if _, ok := store.vehicles[vehicle.ID()]; !ok {
		return application.ErrVehicleNotFound
	}
	store.vehicles[vehicle.ID()] = vehicle
	return nil
}

func (store *handlerVehicleStore) Delete(_ context.Context, id string) error {
	if _, ok := store.vehicles[id]; !ok {
		return application.ErrVehicleNotFound
	}
	delete(store.vehicles, id)
	return nil
}

type handlerCustomerDirectory struct{}

func (handlerCustomerDirectory) IsActiveCustomer(_ context.Context, id string) (bool, error) {
	return id == "customer-1", nil
}

type handlerIDGenerator struct{}

func (handlerIDGenerator) NewID(context.Context) (string, error) { return "vehicle-1", nil }

func newTestEcho() *echo.Echo {
	store := &handlerVehicleStore{vehicles: make(map[string]domain.Vehicle)}
	service := application.NewVehicleService(store, handlerCustomerDirectory{}, handlerIDGenerator{})
	e := echo.New()
	NewHandler(service).RegisterRoutes(e.Group("/api/v1").Group("/vehicles"))
	return e
}

func serve(e *echo.Echo, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

// failingListStore fails every List call with err.
type failingListStore struct {
	*handlerVehicleStore
	err error
}

func (store failingListStore) List(context.Context, application.ListFilter, pagination.Params) (pagination.Page[domain.Vehicle], error) {
	return pagination.Page[domain.Vehicle]{}, store.err
}

func newTestEchoWithStore(store application.VehicleStore) *echo.Echo {
	e := echo.New()
	service := application.NewVehicleService(store, handlerCustomerDirectory{}, handlerIDGenerator{})
	NewHandler(service).RegisterRoutes(e.Group("/api/v1").Group("/vehicles"))
	return e
}
