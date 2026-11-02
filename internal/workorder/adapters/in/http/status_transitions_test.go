package workorderhttp

import (
	"net/http"
	"testing"

	"github.com/medanielsantos/fiap-soat/internal/workorder/domain"
)

const base = "/api/v1/work-orders/id-1"

func diagnosedWithPart(t *testing.T) *testServer {
	t.Helper()
	s := newTestServer()
	s.mustDo(t, http.MethodPost, "/api/v1/work-orders", createBody, http.StatusCreated)
	s.mustDo(t, http.MethodPost, base+"/diagnosis", `{"diagnosis":"brake noise"}`, http.StatusOK)
	s.mustDo(t, http.MethodPost, base+"/items", `{"type":"part","catalog_id":"part-1","quantity":2}`, http.StatusCreated)
	return s
}

func TestWorkOrderHappyPath(t *testing.T) {
	s := diagnosedWithPart(t)
	order := s.mustDo(t, http.MethodPost, base+"/items", `{"type":"service","catalog_id":"svc-1","quantity":1}`, http.StatusCreated)
	if len(order.Items) != 2 || order.Diagnosis != "brake noise" {
		t.Fatalf("order = %+v, want two items and diagnosis", order)
	}
	steps := []struct {
		path, body string
		want       domain.Status
	}{
		{"/quotes", `{}`, domain.AwaitingApproval},
		{"/quotes/approve", `{"client_id":"client-1"}`, domain.InProgress},
		{"/consumptions", `{"part_id":"part-1","quantity":2}`, domain.InProgress},
		{"/complete", `{}`, domain.Completed},
		{"/deliver", `{}`, domain.Delivered},
	}
	for _, step := range steps {
		order = s.mustDo(t, http.MethodPost, base+step.path, step.body, http.StatusOK)
		if order.Status != step.want {
			t.Fatalf("after %s status = %s, want %s", step.path, order.Status, step.want)
		}
	}
	if order.UpdatedBy != "user-1" {
		t.Errorf("updated_by = %q, want authenticated user", order.UpdatedBy)
	}
}

func TestRemoveItem(t *testing.T) {
	s := diagnosedWithPart(t)
	if order := s.mustDo(t, http.MethodDelete, base+"/items/id-4", "", http.StatusOK); len(order.Items) != 0 {
		t.Fatalf("items = %+v, want none", order.Items)
	}
	s.mustDo(t, http.MethodDelete, base+"/items/id-4", "", http.StatusConflict)
	s.mustDo(t, http.MethodDelete, "/api/v1/work-orders/missing/items/id-4", "", http.StatusNotFound)
}

func TestInvalidTransitionsReturnConflict(t *testing.T) {
	s := newTestServer()
	s.mustDo(t, http.MethodPost, "/api/v1/work-orders", createBody, http.StatusCreated)
	for _, path := range []string{"/complete", "/deliver", "/quotes", "/quotes/approve"} {
		body := `{}`
		if path == "/quotes/approve" {
			body = `{"client_id":"client-1"}`
		}
		response := s.do(http.MethodPost, base+path, body)
		if response.Code != http.StatusConflict {
			t.Errorf("POST %s status = %d, want 409", path, response.Code)
		}
	}
	s.mustDo(t, http.MethodPost, base+"/consumptions", `{"part_id":"part-1","quantity":1}`, http.StatusConflict)
	s.mustDo(t, http.MethodPost, base+"/items", `{"type":"part","catalog_id":"part-1","quantity":1}`, http.StatusConflict)
}

func TestValidationErrorsReturnBadRequest(t *testing.T) {
	s := diagnosedWithPart(t)
	if p := decodeProblem(t, s.do(http.MethodPost, base+"/diagnosis", `{"diagnosis":"  "}`)); p.Status != http.StatusBadRequest || p.Title != "Invalid work order" {
		t.Errorf("blank diagnosis problem = %+v", p)
	}
	s.mustDo(t, http.MethodPost, base+"/items", `{"type":"part","catalog_id":"part-9","quantity":1}`, http.StatusUnprocessableEntity)
	s.mustDo(t, http.MethodPost, base+"/quotes", `{}`, http.StatusOK)
	s.mustDo(t, http.MethodPost, base+"/quotes/approve", `{"client_id":""}`, http.StatusBadRequest)
	s.mustDo(t, http.MethodPost, base+"/quotes/approve", `{"client_id":"client-1"}`, http.StatusOK)
	s.mustDo(t, http.MethodPost, base+"/consumptions", `{"part_id":"part-1","quantity":0}`, http.StatusBadRequest)
	s.mustDo(t, http.MethodPost, base+"/consumptions", `{"part_id":"part-1","quantity":5}`, http.StatusConflict)
	s.mustDo(t, http.MethodPost, base+"/complete", `{}`, http.StatusConflict)
}

func TestMalformedBodiesReturnBadRequest(t *testing.T) {
	s := diagnosedWithPart(t)
	for _, path := range []string{"/diagnosis", "/items", "/quotes", "/quotes/approve", "/quotes/reject", "/consumptions", "/complete", "/deliver"} {
		response := s.do(http.MethodPost, base+path, `{"unexpected":true}`)
		if response.Code != http.StatusBadRequest {
			t.Errorf("POST %s status = %d, want 400", path, response.Code)
		}
	}
}
