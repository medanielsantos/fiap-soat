package catalog

import (
	"context"
	"errors"
	"testing"

	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	catalogdomain "github.com/medanielsantos/fiap-soat/internal/servicecatalog/domain"
	"github.com/medanielsantos/fiap-soat/internal/workorder/application"
)

func TestDirectoryService(t *testing.T) {
	service, err := catalogdomain.NewService("svc-1", "ALIGN", "Alignment", "Wheel alignment", 5000, 45, true, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	directory := NewDirectory(newCatalog(&stubStore{service: service}))
	got, err := directory.Service(context.Background(), "svc-1")
	if err != nil {
		t.Fatalf("Service() error = %v", err)
	}
	want := application.CatalogItem{ID: "svc-1", Code: "ALIGN", Name: "Alignment", Description: "Wheel alignment", Unit: "service", SalePriceInCents: 5000, EstimatedDurationMinutes: 45, Active: true}
	if got != want {
		t.Errorf("Service() = %+v, want %+v", got, want)
	}

	failing := NewDirectory(newCatalog(&stubStore{findErr: catalogapp.ErrNotFound}))
	if _, err := failing.Service(context.Background(), "svc-1"); !errors.Is(err, catalogapp.ErrNotFound) {
		t.Errorf("Service() error = %v, want %v", err, catalogapp.ErrNotFound)
	}
}

func TestDirectoryPart(t *testing.T) {
	part, err := catalogdomain.NewPart("part-1", "SKU-1", "PAD", "Brake pad", "Front", "pair", 1000, 2500, 10, 0, false, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	directory := NewDirectory(newCatalog(&stubStore{part: part}))
	got, err := directory.Part(context.Background(), "part-1")
	if err != nil {
		t.Fatalf("Part() error = %v", err)
	}
	want := application.CatalogItem{ID: "part-1", Code: "SKU-1", Name: "Brake pad", Description: "Front", Unit: "pair", SalePriceInCents: 2500, Active: false}
	if got != want {
		t.Errorf("Part() = %+v, want %+v (SKU as code, inactive preserved)", got, want)
	}

	failing := NewDirectory(newCatalog(&stubStore{findErr: catalogapp.ErrNotFound}))
	if _, err := failing.Part(context.Background(), "part-1"); !errors.Is(err, catalogapp.ErrNotFound) {
		t.Errorf("Part() error = %v, want %v", err, catalogapp.ErrNotFound)
	}
}
