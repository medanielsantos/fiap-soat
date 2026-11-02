package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

func seedOrders(t *testing.T, repo *Repository) {
	t.Helper()
	specs := []orderSpec{
		{id: "wo-1", customerID: "customer-1", vehicleID: "vehicle-1"},
		{id: "wo-2", customerID: "customer-1", vehicleID: "vehicle-2"},
		{id: "wo-3", customerID: "customer-2", vehicleID: "vehicle-3"},
	}
	for i, spec := range specs {
		spec.createdAt = baseTime.Add(time.Duration(i) * time.Hour)
		createOrder(t, repo, spec)
	}
	o, err := repo.Find(context.Background(), "wo-2")
	if err != nil {
		t.Fatal(err)
	}
	apply(t, repo, &o, "diagnosis", func(o *domain.WorkOrder) (domain.HistoryEntry, error) {
		return o.SetDiagnosis("noise", "user-1", baseTime)
	})
}

func ids(orders []domain.WorkOrder) []string {
	out := make([]string, 0, len(orders))
	for _, o := range orders {
		out = append(out, o.ID)
	}
	return out
}

func params(t *testing.T, page, perPage int) pagination.Params {
	t.Helper()
	p, err := pagination.NewParams(page, perPage)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRepositoryList(t *testing.T) {
	repo, _ := openRepository(t)
	seedOrders(t, repo)

	tests := []struct {
		name      string
		status    string
		page, per int
		want      []string
		total     int64
	}{
		{name: "test_it_should_list_newest_first", page: 1, per: 10, want: []string{"wo-3", "wo-2", "wo-1"}, total: 3},
		{name: "test_it_should_paginate", page: 2, per: 2, want: []string{"wo-1"}, total: 3},
		{name: "test_it_should_filter_by_status", status: "received", page: 1, per: 10, want: []string{"wo-3", "wo-1"}, total: 2},
		{name: "test_it_should_return_empty_page_for_unused_status", status: "delivered", page: 1, per: 10, want: []string{}, total: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, err := repo.List(context.Background(), test.status, params(t, test.page, test.per))
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if got := ids(page.Items); page.TotalItems != test.total || len(got) != len(test.want) || (len(got) > 0 && got[0] != test.want[0]) {
				t.Errorf("List() = %v (total %d), want %v (total %d)", got, page.TotalItems, test.want, test.total)
			}
		})
	}
}

func TestRepositoryListForClient(t *testing.T) {
	repo, _ := openRepository(t)
	seedOrders(t, repo)

	page, err := repo.ListForClient(context.Background(), "customer-1", "", params(t, 1, 10))
	if err != nil {
		t.Fatalf("ListForClient() error = %v", err)
	}
	if got := ids(page.Items); page.TotalItems != 2 || got[0] != "wo-2" || got[1] != "wo-1" {
		t.Errorf("ListForClient(customer-1) = %v, want [wo-2 wo-1]", got)
	}
	if len(page.Items[0].History) != 2 {
		t.Errorf("listed order history = %d events, want loaded history", len(page.Items[0].History))
	}

	page, err = repo.ListForClient(context.Background(), "customer-1", "vehicle-1", params(t, 1, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(page.Items); page.TotalItems != 1 || got[0] != "wo-1" {
		t.Errorf("ListForClient(customer-1, vehicle-1) = %v, want [wo-1]", got)
	}
}
