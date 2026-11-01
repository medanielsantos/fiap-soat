package integration_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v5"
	customerpostgres "github.com/medanielsantos/fiap-soat/internal/customer/adapters/out/postgres"
	customerapp "github.com/medanielsantos/fiap-soat/internal/customer/application"
	"github.com/medanielsantos/fiap-soat/internal/platform/config"
	"github.com/medanielsantos/fiap-soat/internal/platform/database"
	platformid "github.com/medanielsantos/fiap-soat/internal/platform/id"
	cataloghttp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/in/http"
	catalogpostgres "github.com/medanielsantos/fiap-soat/internal/servicecatalog/adapters/out/postgres"
	catalogapp "github.com/medanielsantos/fiap-soat/internal/servicecatalog/application"
	"github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/out/customer"
	vehiclepostgres "github.com/medanielsantos/fiap-soat/internal/vehicle/adapters/out/postgres"
	vehicleapp "github.com/medanielsantos/fiap-soat/internal/vehicle/application"
	workorderhttp "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/in/http"
	workordercatalog "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/out/catalog"
	workorderownership "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/out/customervehicle"
	workorderpostgres "github.com/medanielsantos/fiap-soat/internal/workorder/adapters/out/postgres"
	workorderapp "github.com/medanielsantos/fiap-soat/internal/workorder/application"
	"github.com/medanielsantos/fiap-soat/migrations"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testCustomerID = "integration-customer"
	testClientID   = testCustomerID
	testVehicleID  = "integration-vehicle"
	testUserID     = "integration-user"
)

type testServer struct {
	*echo.Echo
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	config.LoadDotEnvFile("../../.env.testing")
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = config.PostgresURLFromEnvironment()
	}
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL or add DB_* credentials to .env.testing to run integration tests")
	}

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin.SetMaxOpenConns(1)
	if err := admin.PingContext(context.Background()); err != nil {
		if closeErr := admin.Close(); closeErr != nil {
			t.Errorf("close admin connection: %v", closeErr)
		}
		t.Fatalf("connect to TEST_DATABASE_URL: %v", err)
	}
	schema := fmt.Sprintf("integration_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA "` + schema + `"`); err != nil {
		if closeErr := admin.Close(); closeErr != nil {
			t.Errorf("close admin connection: %v", closeErr)
		}
		t.Fatalf("create isolated test schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`); err != nil {
			t.Errorf("drop isolated test schema: %v", err)
		}
		_ = admin.Close()
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema+",public")
	parsed.RawQuery = query.Encode()
	schemaDSN := parsed.String()
	sqlDB, err := sql.Open("pgx", schemaDSN)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(10)
	t.Cleanup(func() { _ = sqlDB.Close() })
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("apply test migrations: %v", err)
	}
	db, err := gorm.Open(postgres.Open(schemaDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := seedOwnerRecords(db); err != nil {
		t.Fatalf("seed customer and vehicle: %v", err)
	}

	customerRepo := customerpostgres.NewCustomerRepository(db)
	customerService := customerapp.NewCustomerService(customerRepo, platformid.UUIDGenerator{})
	vehicleRepo := vehiclepostgres.NewVehicleRepository(db)
	vehicleService := vehicleapp.NewVehicleService(vehicleRepo, customer.NewDirectory(customerService), platformid.UUIDGenerator{})
	catalogRepo := catalogpostgres.NewRepository(db)
	catalogService := catalogapp.NewCatalogService(catalogRepo, platformid.UUIDGenerator{})
	tx := database.NewTransactionManager(db)
	workRepo := workorderpostgres.NewRepository(db)
	ownership := workorderownership.NewDirectory(customerService, vehicleService)
	workCatalog := workordercatalog.NewDirectory(catalogService)
	inventory := workordercatalog.NewInventory(catalogService)
	workService := workorderapp.NewService(workRepo, tx, platformid.UUIDGenerator{}, ownership, workCatalog, inventory)
	engine := echo.New()
	v1 := engine.Group("/api/v1")
	cataloghttp.NewHandler(catalogService).RegisterRoutes(v1)
	workorderhttp.NewHandler(workService).RegisterRoutes(v1.Group("/work-orders"))
	return &testServer{Echo: engine}
}

func seedOwnerRecords(db *gorm.DB) error {
	if err := db.Exec(`INSERT INTO customers (id, document, name, email, phone, status) VALUES (?, ?, ?, ?, ?, ?)`, testCustomerID, "52998224725", "Integration Customer", "integration@example.test", "+5511999999999", "active").Error; err != nil {
		return err
	}
	return db.Exec(`INSERT INTO vehicles (id, customer_id, plate, brand, model, year) VALUES (?, ?, ?, ?, ?, ?)`, testVehicleID, testCustomerID, "ABC1D23", "Test", "Vehicle", time.Now().Year()).Error
}

func createPart(t *testing.T, server *testServer, sku string, initial int64) string {
	t.Helper()
	body := map[string]any{"sku": sku, "internal_code": sku, "name": "Brake pad", "unit": "piece", "purchase_price_in_cents": 100, "sale_price_in_cents": 250, "initial_stock": initial, "created_by": testUserID}
	return call(t, server, http.MethodPost, "/api/v1/parts", body, http.StatusCreated)["id"].(string)
}

func createOrder(t *testing.T, server *testServer) string {
	t.Helper()
	body := map[string]any{"customer_id": testCustomerID, "vehicle_id": testVehicleID, "user_id": testUserID}
	return call(t, server, http.MethodPost, "/api/v1/work-orders", body, http.StatusCreated)["id"].(string)
}

func prepareOrder(t *testing.T, server *testServer, partID string, quantity int64) string {
	t.Helper()
	id := createOrder(t, server)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+id+"/diagnosis", map[string]any{"diagnosis": "Replace part", "user_id": testUserID}, http.StatusOK)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+id+"/items", map[string]any{"type": "part", "catalog_id": partID, "quantity": quantity, "user_id": testUserID}, http.StatusCreated)
	call(t, server, http.MethodPost, "/api/v1/work-orders/"+id+"/quotes", map[string]any{"user_id": testUserID}, http.StatusOK)
	return id
}

func assertStock(t *testing.T, server *testServer, partID string, stock, reserved int64) {
	t.Helper()
	part := call(t, server, http.MethodGet, "/api/v1/parts/"+partID, nil, http.StatusOK)
	if part["stock"] != float64(stock) || part["reserved_stock"] != float64(reserved) {
		t.Fatalf("stock = %v, reserved = %v; want %d and %d", part["stock"], part["reserved_stock"], stock, reserved)
	}
}

func call(t *testing.T, server *testServer, method, path string, body any, wantStatus int) map[string]any {
	t.Helper()
	var requestBody []byte
	if body != nil {
		var err error
		requestBody, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewReader(requestBody))
	server.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s returned %d, want %d: %s", method, path, recorder.Code, wantStatus, strings.TrimSpace(recorder.Body.String()))
	}
	if recorder.Body.Len() == 0 {
		return map[string]any{}
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode %s %s response: %v (%s)", method, path, err, recorder.Body.String())
	}
	return response
}
