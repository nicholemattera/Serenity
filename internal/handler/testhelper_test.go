package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicholemattera/serenity/internal/database"
	"github.com/nicholemattera/serenity/internal/handler"
	"github.com/nicholemattera/serenity/internal/repository"
	"github.com/nicholemattera/serenity/internal/service"
)

// testServer holds the router and all services for E2E tests.
type testServer struct {
	router        http.Handler
	roleSvc       service.RoleService
	userSvc       service.UserService
	permissionSvc service.PermissionService
	authSvc       service.AuthService
	compositeSvc  service.CompositeService
	fieldSvc      service.FieldService
	entitySvc     service.EntityService
	fieldValueSvc service.FieldValueService
}

func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("serenity_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Errorf("failed to terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := database.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := database.Migrate(pool); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return pool
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	db := newTestDB(t)

	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	compositeRepo := repository.NewCompositeRepository(db)
	fieldRepo := repository.NewFieldRepository(db)
	entityRepo := repository.NewEntityRepository(db)
	fieldValueRepo := repository.NewFieldValueRepository(db)

	roleSvc := service.NewRoleService(roleRepo)
	userSvc := service.NewUserService(userRepo, 4) // low bcrypt cost for tests
	permissionSvc := service.NewPermissionService(permissionRepo, 45*time.Second, 1000)
	authSvc := service.NewAuthService(userRepo, roleRepo, "test-secret", "serenity", "serenity", 4) // low bcrypt cost for tests
	fieldSvc := service.NewFieldService(fieldRepo)
	compositeSvc := service.NewCompositeService(compositeRepo, fieldSvc)
	fieldValueSvc := service.NewFieldValueService(fieldValueRepo, fieldSvc)
	entitySvc := service.NewEntityService(entityRepo, fieldValueSvc)

	authHandler := handler.NewAuthHandler(authSvc, userSvc, roleSvc)
	roleHandler := handler.NewRoleHandler(roleSvc, permissionSvc)
	permissionHandler := handler.NewPermissionHandler(permissionSvc)
	userHandler := handler.NewUserHandler(userSvc, roleSvc, permissionSvc)
	compositeHandler := handler.NewCompositeHandler(compositeSvc, permissionSvc)
	fieldHandler := handler.NewFieldHandler(fieldSvc, permissionSvc)
	entityHandler := handler.NewEntityHandler(entitySvc, fieldSvc, fieldValueSvc, compositeSvc, permissionSvc)
	fieldValueHandler := handler.NewFieldValueHandler(fieldValueSvc, entitySvc, compositeSvc, permissionSvc)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Route("/v1", func(v1Router chi.Router) {
		v1Router.Use(handler.Authenticate(authSvc))

		v1Router.Post("/auth/login", authHandler.Login)
		v1Router.Post("/auth/register", authHandler.Register)

		v1Router.With(handler.RequireAuth).Get("/roles", roleHandler.List)
		v1Router.With(handler.RequireAuth).Post("/roles", roleHandler.Create)
		v1Router.With(handler.RequireAuth).Get("/roles/{id}", roleHandler.GetByID)
		v1Router.With(handler.RequireAuth).Put("/roles/{id}", roleHandler.Update)
		v1Router.With(handler.RequireAuth).Delete("/roles/{id}", roleHandler.Delete)
		v1Router.With(handler.RequireAuth).Get("/roles/{roleID}/permissions", permissionHandler.ListByRole)

		v1Router.With(handler.RequireAuth).Post("/permissions", permissionHandler.Create)
		v1Router.With(handler.RequireAuth).Get("/permissions/{id}", permissionHandler.GetByID)
		v1Router.With(handler.RequireAuth).Put("/permissions/{id}", permissionHandler.Update)
		v1Router.With(handler.RequireAuth).Delete("/permissions/{id}", permissionHandler.Delete)

		v1Router.With(handler.RequireAuth).Get("/users", userHandler.List)
		v1Router.With(handler.RequireAuth).Post("/users", userHandler.Create)
		v1Router.With(handler.RequireAuth).Get("/users/{id}", userHandler.GetByID)
		v1Router.With(handler.RequireAuth).Put("/users/{id}", userHandler.Update)
		v1Router.With(handler.RequireAuth).Put("/users/{id}/password", userHandler.UpdatePassword)
		v1Router.With(handler.RequireAuth).Delete("/users/{id}", userHandler.Delete)

		v1Router.With(handler.RequireAuth).Get("/composites", compositeHandler.List)
		v1Router.With(handler.RequireAuth).Post("/composites", compositeHandler.Create)
		v1Router.With(handler.RequireAuth).Get("/composites/slug/{slug}", compositeHandler.GetBySlug)
		v1Router.With(handler.RequireAuth).Get("/composites/{id}", compositeHandler.GetByID)
		v1Router.With(handler.RequireAuth).Put("/composites/{id}", compositeHandler.Update)
		v1Router.With(handler.RequireAuth).Delete("/composites/{id}", compositeHandler.Delete)

		v1Router.With(handler.RequireAuth).Get("/composites/{compositeID}/fields", fieldHandler.ListByComposite)
		v1Router.With(handler.RequireAuth).Post("/fields", fieldHandler.Create)
		v1Router.With(handler.RequireAuth).Get("/fields/{id}", fieldHandler.GetByID)
		v1Router.With(handler.RequireAuth).Put("/fields/{id}", fieldHandler.Update)
		v1Router.With(handler.RequireAuth).Delete("/fields/{id}", fieldHandler.Delete)

		v1Router.Get("/composites/{compositeID}/entities", entityHandler.ListByComposite)
		v1Router.Get("/composites/{compositeID}/entities/slug/{slug}", entityHandler.GetBySlug)
		v1Router.Post("/entities", entityHandler.Create)
		v1Router.Get("/entities/{id}", entityHandler.GetByID)
		v1Router.With(handler.RequireAuth).Put("/entities/{id}", entityHandler.Update)
		v1Router.With(handler.RequireAuth).Delete("/entities/{id}", entityHandler.Delete)
		v1Router.Get("/entities/{id}/children", entityHandler.ListChildren)
		v1Router.Post("/entities/{id}/move", entityHandler.Move)
		v1Router.Post("/entities/{id}/move-root", entityHandler.MoveRoot)

		v1Router.Get("/entities/{entityID}/field-values", fieldValueHandler.ListByEntity)
		v1Router.Post("/field-values", fieldValueHandler.Set)
		v1Router.Get("/field-values/{id}", fieldValueHandler.GetByID)
		v1Router.With(handler.RequireAuth).Delete("/field-values/{id}", fieldValueHandler.Delete)
	})

	return &testServer{
		router:        r,
		roleSvc:       roleSvc,
		userSvc:       userSvc,
		permissionSvc: permissionSvc,
		authSvc:       authSvc,
		compositeSvc:  compositeSvc,
		fieldSvc:      fieldSvc,
		entitySvc:     entitySvc,
		fieldValueSvc: fieldValueSvc,
	}
}

// do executes a request against the test server.
func (s *testServer) do(method, path string, body any, token string) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			panic(err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	return rr
}

// decode unmarshals the response body into v.
func decode(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode response: %v (body: %s)", err, rr.Body.String())
	}
}

// assertStatus fails the test if the response code does not match.
func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rr.Code != expected {
		t.Fatalf("expected status %d, got %d (body: %s)", expected, rr.Code, rr.Body.String())
	}
}
