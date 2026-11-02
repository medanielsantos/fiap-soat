package application

import (
	"context"
	"errors"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
)

var validService = ServiceInput{InternalCode: "S-1", Name: "Alignment", SalePriceInCents: 5000, EstimatedDurationMinutes: 30, Active: true, PerformedBy: "admin"}

func TestCatalogServiceCreateService(t *testing.T) {
	t.Run("test_it_should_persist_service_with_generated_id", func(t *testing.T) {
		svc, store := newTestService(0)
		got, err := svc.CreateService(context.Background(), validService)
		if err != nil {
			t.Fatalf("CreateService() error = %v", err)
		}
		if got.ID() != "id-1" || got.PerformedBy() != "admin" {
			t.Errorf("service = %+v, want id-1 created by admin", got)
		}
		if _, ok := store.services["id-1"]; !ok {
			t.Error("service was not saved")
		}
	})

	tests := []struct {
		name    string
		failID  int
		input   ServiceInput
		saveErr error
		want    error
	}{
		{"id_generation_fails", 1, validService, nil, errBoom},
		{"price_is_invalid", 0, ServiceInput{InternalCode: "S", Name: "N", EstimatedDurationMinutes: 1}, nil, domain.ErrInvalidPrice},
		{"store_rejects_duplicate", 0, validService, ErrDuplicate, ErrDuplicate},
	}
	for _, tt := range tests {
		t.Run("test_it_should_fail_when_"+tt.name, func(t *testing.T) {
			svc, store := newTestService(tt.failID)
			store.errs["SaveService"] = tt.saveErr
			if _, err := svc.CreateService(context.Background(), tt.input); !errors.Is(err, tt.want) {
				t.Errorf("CreateService() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCatalogServiceReadAndDeleteService(t *testing.T) {
	svc, _ := newTestService(0)
	created, err := svc.CreateService(context.Background(), validService)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := svc.GetService(context.Background(), created.ID()); err != nil || got.Name() != "Alignment" {
		t.Errorf("GetService() = %+v, %v", got, err)
	}
	page, err := svc.ListServices(context.Background(), defaultParams(t))
	if err != nil || page.TotalItems != 1 {
		t.Errorf("ListServices() total = %d, err = %v, want 1", page.TotalItems, err)
	}
	if err := svc.DeleteService(context.Background(), created.ID()); err != nil {
		t.Fatalf("DeleteService() error = %v", err)
	}
	if err := svc.DeleteService(context.Background(), created.ID()); !errors.Is(err, ErrNotFound) {
		t.Errorf("second DeleteService() error = %v, want %v", err, ErrNotFound)
	}
}

func TestCatalogServiceUpdateService(t *testing.T) {
	t.Run("test_it_should_replace_fields_when_service_exists", func(t *testing.T) {
		svc, store := newTestService(0)
		created, _ := svc.CreateService(context.Background(), validService)
		in := validService
		in.Name, in.Active = "Balancing", false
		got, err := svc.UpdateService(context.Background(), created.ID(), in)
		if err != nil {
			t.Fatalf("UpdateService() error = %v", err)
		}
		if got.Name() != "Balancing" || got.Active() || store.services[created.ID()].Name() != "Balancing" {
			t.Errorf("updated service = %+v", got)
		}
	})

	t.Run("test_it_should_return_not_found_when_service_is_missing", func(t *testing.T) {
		svc, _ := newTestService(0)
		if _, err := svc.UpdateService(context.Background(), "missing", validService); !errors.Is(err, ErrNotFound) {
			t.Errorf("UpdateService() error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("test_it_should_reject_invalid_duration", func(t *testing.T) {
		svc, _ := newTestService(0)
		created, _ := svc.CreateService(context.Background(), validService)
		in := validService
		in.EstimatedDurationMinutes = 0
		if _, err := svc.UpdateService(context.Background(), created.ID(), in); !errors.Is(err, domain.ErrInvalidDuration) {
			t.Errorf("UpdateService() error = %v, want %v", err, domain.ErrInvalidDuration)
		}
	})
}
