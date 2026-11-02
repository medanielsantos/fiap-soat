package customerhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/customer/domain"
	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	workorderapp "github.com/medanielsantos/fiap-soat/internal/workorder/application"
	workorderdomain "github.com/medanielsantos/fiap-soat/internal/workorder/domain"
	"golang.org/x/crypto/bcrypt"
)

const clientPassword = "correct-horse-battery"

type clientEnv struct {
	echo       *echo.Echo
	customers  *clientCustomerStore
	limiter    *fakeLimiter
	vehicles   *fakeVehicleStore
	workOrders *fakeWorkOrderStore
	inventory  *fakeInventory
	issuer     *identityjwt.Issuer
	mailer     *fakeMailer
}

// newClientEnv wires admin and client routes with an active customer c-1 and an inactive c-2.
func newClientEnv(t *testing.T) *clientEnv {
	t.Helper()
	env := &clientEnv{
		customers: newClientCustomerStore(), limiter: &fakeLimiter{}, vehicles: &fakeVehicleStore{},
		workOrders: &fakeWorkOrderStore{orders: map[string]workorderdomain.WorkOrder{}}, mailer: &fakeMailer{}, inventory: &fakeInventory{},
	}
	issuer, err := identityjwt.NewIssuer(strings.Repeat("s", 32), "fiap-test", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	env.issuer = issuer
	env.addCustomer(t, "c-1", "529.982.247-25", "ana@example.com", domain.CustomerStatusActive)
	env.addCustomer(t, "c-2", "11.222.333/0001-81", "co@example.com", domain.CustomerStatusInactive)

	service := application.NewCustomerService(env.customers, handlerIDGenerator{}, application.CustomerResetOptions{Mailer: env.mailer, URL: "https://app.example.com/reset"})
	handler := NewHandler(service, ClientDependencies{
		Vehicles:    vehicleapp.NewVehicleService(env.vehicles, nil, nil),
		Issuer:      issuer,
		WorkOrders:  workorderapp.NewService(env.workOrders, inlineTx{}, &sequenceIDs{}, nil, nil, env.inventory),
		RateLimiter: env.limiter,
	})
	env.echo = echo.New()
	group := env.echo.Group("/api/v1").Group("/customers")
	handler.RegisterRoutes(group)
	handler.RegisterClientRoutes(group)
	return env
}

func (env *clientEnv) addCustomer(t *testing.T, id, document, email string, status domain.CustomerStatus) {
	t.Helper()
	doc, err := domain.NewDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(clientPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	customer, err := domain.RestoreCustomerWithPassword(id, doc, "Customer "+id, email, "123", status, string(hash))
	if err != nil {
		t.Fatal(err)
	}
	env.customers.customers[id] = customer
}

func (env *clientEnv) token(t *testing.T, customerID, vehicleID string) string {
	t.Helper()
	token, _, err := env.issuer.IssueClient(customerID, vehicleID)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (env *clientEnv) do(method, path, body, token string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		request.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	response := httptest.NewRecorder()
	env.echo.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, want, response.Body.String())
	}
}

func assertProblem(t *testing.T, response *httptest.ResponseRecorder, status int, title string) {
	t.Helper()
	assertStatus(t, response, status)
	if got := response.Header().Get(echo.HeaderContentType); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	var body problem
	// Decode only the first JSON value: assertHandlerStopped checks there are no trailing writes.
	if err := json.NewDecoder(bytes.NewReader(response.Body.Bytes())).Decode(&body); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if body.Title != title || body.Status != status {
		t.Errorf("problem = %+v, want title %q status %d", body, title, status)
	}
}

func decodeJSON[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, response.Body.String())
	}
	return out
}

// assertHandlerStopped fails when a second JSON value follows the problem body, i.e. the handler
// kept running after the rate limiter rejected the request.
func assertHandlerStopped(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(response.Body.Bytes()))
	var first, second json.RawMessage
	if decoder.Decode(&first) == nil && decoder.Decode(&second) == nil {
		t.Errorf("handler kept running after rate-limit rejection, extra body %s", second)
	}
}
