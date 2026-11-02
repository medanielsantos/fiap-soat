package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/vehicle/application"
)

// TestVehicleRepositorySurfacesDatabaseErrors breaks the schema and checks that
// unexpected database errors are returned untranslated instead of masked as domain errors.
func TestVehicleRepositorySurfacesDatabaseErrors(t *testing.T) {
	ops := map[string]func(*VehicleRepository, *testing.T) error{
		"save": func(r *VehicleRepository, t *testing.T) error {
			return r.Save(context.Background(), newVehicle(t, "vehicle-2", "customer-1", "DEF4G56"))
		},
		"find": func(r *VehicleRepository, _ *testing.T) error {
			_, err := r.FindByID(context.Background(), "vehicle-1")
			return err
		},
		"list": func(r *VehicleRepository, t *testing.T) error {
			_, err := r.List(context.Background(), application.ListFilter{}, firstPage(t, 10))
			return err
		},
		"update": func(r *VehicleRepository, t *testing.T) error {
			return r.Update(context.Background(), newVehicle(t, "vehicle-1", "customer-1", "ABC1D23"))
		},
		"delete": func(r *VehicleRepository, _ *testing.T) error {
			return r.Delete(context.Background(), "vehicle-1")
		},
	}
	schemaBreaks := map[string]string{
		"dropped_table":  "DROP TABLE vehicles CASCADE",
		"renamed_column": "ALTER TABLE vehicles RENAME COLUMN created_at TO registered_at",
	}
	for breakName, ddl := range schemaBreaks {
		for opName, op := range ops {
			if breakName == "renamed_column" && opName != "list" {
				continue // only the ordered list query depends on created_at by name
			}
			t.Run(breakName+"_"+opName, func(t *testing.T) {
				repository, db := openRepository(t)
				if err := repository.Save(context.Background(), newVehicle(t, "vehicle-1", "customer-1", "ABC1D23")); err != nil {
					t.Fatal(err)
				}
				if err := db.Exec(ddl).Error; err != nil {
					t.Fatalf("break schema: %v", err)
				}
				err := op(repository, t)
				if err == nil || errors.Is(err, application.ErrVehicleNotFound) || errors.Is(err, application.ErrDuplicatePlate) {
					t.Errorf("error = %v, want raw database error", err)
				}
			})
		}
	}
}
