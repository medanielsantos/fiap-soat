package customerhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	vehicledomain "github.com/medanielsantos/fiap-soat/internal/vehicle/domain"
)

func TestClientVehicles(t *testing.T) {
	newVehicle := func(t *testing.T) vehicledomain.Vehicle {
		plate, err := vehicledomain.NewPlate("ABC1D23")
		if err != nil {
			t.Fatal(err)
		}
		vehicle, err := vehicledomain.NewVehicle("v-1", "c-1", plate, "Fiat", "Uno", 2010)
		if err != nil {
			t.Fatal(err)
		}
		return vehicle
	}

	t.Run("test_it_should_list_vehicles_scoped_to_token_customer_and_vehicle", func(t *testing.T) {
		env := newClientEnv(t)
		env.vehicles.vehicles = []vehicledomain.Vehicle{newVehicle(t)}
		response := env.do(http.MethodGet, vehiclesPath+"?page=1&per_page=5", "", env.token(t, "c-1", "v-1"))
		assertStatus(t, response, http.StatusOK)
		page := decodeJSON[pagination.Response[clientVehicle]](t, response)
		if len(page.Data) != 1 || page.Data[0] != (clientVehicle{ID: "v-1", Plate: "ABC1D23", Brand: "Fiat", Model: "Uno", Year: 2010}) {
			t.Errorf("data = %+v, want the seeded vehicle", page.Data)
		}
		if env.vehicles.lastFilter.CustomerID != "c-1" || env.vehicles.lastFilter.VehicleID != "v-1" {
			t.Errorf("filter = %+v, want customer c-1 and vehicle v-1", env.vehicles.lastFilter)
		}
	})

	t.Run("test_it_should_return_bad_request_when_pagination_is_invalid", func(t *testing.T) {
		env := newClientEnv(t)
		assertProblem(t, env.do(http.MethodGet, vehiclesPath+"?page=0", "", env.token(t, "c-1", "")), http.StatusBadRequest, "Invalid pagination")
	})

	t.Run("test_it_should_return_internal_error_when_listing_fails", func(t *testing.T) {
		env := newClientEnv(t)
		env.vehicles.err = errBackend
		assertProblem(t, env.do(http.MethodGet, vehiclesPath, "", env.token(t, "c-1", "")), http.StatusInternalServerError, "Internal server error")
	})
}
