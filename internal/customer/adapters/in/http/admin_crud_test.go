package customerhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/pagination"
)

const customersPath = "/api/v1/customers"

func TestAdminGetAndDelete(t *testing.T) {
	env := newClientEnv(t)

	response := env.do(http.MethodGet, customersPath+"/c-2", "", "")
	assertStatus(t, response, http.StatusOK)
	if got := decodeJSON[customerResponse](t, response); got.Document != "**.***.***/****-81" || got.Status != "inactive" {
		t.Errorf("customer = %+v, want masked CNPJ and inactive status", got)
	}

	assertStatus(t, env.do(http.MethodDelete, customersPath+"/c-2", "", ""), http.StatusNoContent)
	assertProblem(t, env.do(http.MethodGet, customersPath+"/c-2", "", ""), http.StatusNotFound, "Customer not found")
	assertProblem(t, env.do(http.MethodDelete, customersPath+"/c-2", "", ""), http.StatusNotFound, "Customer not found")
}

func TestAdminList(t *testing.T) {
	t.Run("test_it_should_return_paginated_customers", func(t *testing.T) {
		response := newClientEnv(t).do(http.MethodGet, customersPath+"?page=1&per_page=10", "", "")
		assertStatus(t, response, http.StatusOK)
		if page := decodeJSON[pagination.Response[customerResponse]](t, response); len(page.Data) != 2 || page.Pagination.TotalItems != 2 {
			t.Errorf("page = %+v, want two customers", page)
		}
	})

	t.Run("test_it_should_return_bad_request_when_pagination_is_invalid", func(t *testing.T) {
		assertProblem(t, newClientEnv(t).do(http.MethodGet, customersPath+"?per_page=1000", "", ""), http.StatusBadRequest, "Invalid pagination")
	})

	t.Run("test_it_should_return_internal_error_when_store_fails", func(t *testing.T) {
		env := newClientEnv(t)
		env.customers.listErr = errBackend
		assertProblem(t, env.do(http.MethodGet, customersPath, "", ""), http.StatusInternalServerError, "Internal server error")
	})
}

func TestAdminUpdate(t *testing.T) {
	const valid = `{"document":"529.982.247-25","name":"Ana Souza","email":"ana@example.com","phone":"123","status":"active"}`

	t.Run("test_it_should_update_customer", func(t *testing.T) {
		response := newClientEnv(t).do(http.MethodPut, customersPath+"/c-1", valid, "")
		assertStatus(t, response, http.StatusOK)
		if got := decodeJSON[customerResponse](t, response); got.Name != "Ana Souza" || got.ID != "c-1" {
			t.Errorf("customer = %+v, want updated name", got)
		}
	})

	cases := []struct {
		name, id, body string
		storeErr       error
		status         int
		title          string
	}{
		{"test_it_should_return_not_found_for_unknown_customer", "missing", valid, nil, http.StatusNotFound, "Customer not found"},
		{"test_it_should_reject_unknown_fields", "c-1", `{"nickname":"x"}`, nil, http.StatusBadRequest, "Invalid request"},
		{"test_it_should_reject_multiple_json_values", "c-1", valid + valid, nil, http.StatusBadRequest, "Invalid request"},
		{"test_it_should_reject_invalid_status", "c-1", `{"document":"529.982.247-25","name":"A","email":"a@b.co","phone":"1","status":"x"}`, nil, http.StatusBadRequest, "Invalid customer"},
		{"test_it_should_reject_short_password", "c-1", `{"document":"529.982.247-25","name":"A","email":"a@b.co","phone":"1","status":"active","password":"short"}`, nil, http.StatusBadRequest, "Invalid customer"},
		{"test_it_should_map_duplicate_document_to_conflict", "c-1", valid, application.ErrDuplicateDocument, http.StatusConflict, "Customer already exists"},
		{"test_it_should_map_duplicate_email_to_conflict", "c-1", valid, application.ErrDuplicateEmail, http.StatusConflict, "Email already registered"},
		{"test_it_should_hide_unexpected_errors", "c-1", valid, errBackend, http.StatusInternalServerError, "Internal server error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newClientEnv(t)
			env.customers.updateErr = tc.storeErr
			assertProblem(t, env.do(http.MethodPut, customersPath+"/"+tc.id, tc.body, ""), tc.status, tc.title)
		})
	}
}

func TestAdminCreateRejectsMalformedBody(t *testing.T) {
	env := newClientEnv(t)
	for _, body := range []string{`{"name":"A"} {"name":"B"}`, `{"unknown":true}`, `[`} {
		assertProblem(t, env.do(http.MethodPost, customersPath, body, ""), http.StatusBadRequest, "Invalid request")
	}
}

func TestMaskDocument(t *testing.T) {
	cases := map[string]string{"52998224725": "***.***.***-25", "11222333000181": "**.***.***/****-81", "123": ""}
	for input, want := range cases {
		if got := maskDocument(input); got != want {
			t.Errorf("maskDocument(%q) = %q, want %q", input, got, want)
		}
	}
}
