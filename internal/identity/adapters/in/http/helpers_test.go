package identityhttp

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	identityjwt "github.com/medanielsantos/fiap-soat/internal/identity/adapters/out/jwt"
	"github.com/medanielsantos/fiap-soat/internal/identity/application"
)

var allPermissions = []application.Permission{
	application.PermissionUsersCreate, application.PermissionUsersRead, application.PermissionUsersStatusUpdate,
	application.PermissionUsersRoleAssign, application.PermissionUsersPermissionAssign, application.PermissionUsersDelete,
}

type testEnv struct {
	echo   *echo.Echo
	store  *fakeStore
	issuer *identityjwt.Issuer
	mailer *fakeMailer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	issuer, err := identityjwt.NewIssuer("http-test-secret-with-32-bytes-min", "http-test", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	env := &testEnv{echo: echo.New(), store: newFakeStore(), issuer: issuer, mailer: &fakeMailer{}}
	service := application.NewService(env.store, &sequentialIDs{}, issuer, env.mailer, "https://example.test/reset")
	NewHandler(service, issuer).RegisterRoutes(env.echo.Group("/api/v1"))
	return env
}

func (env *testEnv) addUser(t *testing.T, id string, roles []application.UserRole, permissions ...application.Permission) string {
	t.Helper()
	env.store.users[id] = application.User{ID: id, Name: id, Email: id + "@example.test", Status: application.UserStatusActive, Roles: roles, Permissions: permissions}
	token, _, err := env.issuer.Issue(id)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (env *testEnv) admin(t *testing.T) string {
	return env.addUser(t, "admin", []application.UserRole{application.UserRoleAdmin}, allPermissions...)
}

func (env *testEnv) do(method, path, token, body string) *httptest.ResponseRecorder {
	authorization := ""
	if token != "" {
		authorization = "Bearer " + token
	}
	return env.send(method, path, authorization, body)
}

func (env *testEnv) doWithAuthorization(method, path, authorization string) *httptest.ResponseRecorder {
	return env.send(method, path, authorization, "")
}

func (env *testEnv) send(method, path, authorization, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	env.echo.ServeHTTP(response, request)
	return response
}

func assertProblem(t *testing.T, response *httptest.ResponseRecorder, status int, title string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, status, response.Body.String())
	}
	if title == "" {
		return
	}
	var problem map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body: %s", err, response.Body.String())
	}
	if problem["title"] != title || problem["status"] != float64(status) {
		t.Fatalf("problem = %v, want title %q status %d", problem, title, status)
	}
}

func decodeBody[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; body: %s", err, response.Body.String())
	}
	return body
}
