package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/nicholemattera/serenity/internal/config"
	"github.com/nicholemattera/serenity/internal/database"
	"github.com/nicholemattera/serenity/internal/handler"
	"github.com/nicholemattera/serenity/internal/repository"
	"github.com/nicholemattera/serenity/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Repositories
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	compositeRepo := repository.NewCompositeRepository(db)
	fieldRepo := repository.NewFieldRepository(db)
	entityRepo := repository.NewEntityRepository(db)
	fieldValueRepo := repository.NewFieldValueRepository(db)

	// Services
	roleSvc := service.NewRoleService(roleRepo)
	userSvc := service.NewUserService(userRepo, cfg.BCryptCost)
	permissionSvc := service.NewPermissionService(permissionRepo)
	authSvc := service.NewAuthService(userRepo, roleRepo, cfg.JWTSecret)
	fieldSvc := service.NewFieldService(fieldRepo)
	compositeSvc := service.NewCompositeService(compositeRepo, fieldSvc)
	fieldValueSvc := service.NewFieldValueService(fieldValueRepo, fieldSvc)
	entitySvc := service.NewEntityService(entityRepo, fieldValueSvc)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc, userSvc, roleSvc)
	roleHandler := handler.NewRoleHandler(roleSvc, permissionSvc)
	permissionHandler := handler.NewPermissionHandler(permissionSvc)
	userHandler := handler.NewUserHandler(userSvc, permissionSvc)
	compositeHandler := handler.NewCompositeHandler(compositeSvc, permissionSvc)
	fieldHandler := handler.NewFieldHandler(fieldSvc, permissionSvc)
	entityHandler := handler.NewEntityHandler(entitySvc, compositeSvc, permissionSvc)
	fieldValueHandler := handler.NewFieldValueHandler(fieldValueSvc, entitySvc, compositeSvc, permissionSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(handler.Authenticate(authSvc))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Auth
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/register", authHandler.Register)

	// Roles
	r.Get("/roles", roleHandler.List)
	r.Post("/roles", roleHandler.Create)
	r.Get("/roles/{id}", roleHandler.GetByID)
	r.Put("/roles/{id}", roleHandler.Update)
	r.Delete("/roles/{id}", roleHandler.Delete)
	r.Get("/roles/{roleID}/permissions", permissionHandler.ListByRole)

	// Permissions
	r.Post("/permissions", permissionHandler.Create)
	r.Get("/permissions/{id}", permissionHandler.GetByID)
	r.Put("/permissions/{id}", permissionHandler.Update)
	r.Delete("/permissions/{id}", permissionHandler.Delete)

	// Users
	r.Get("/users", userHandler.List)
	r.Post("/users", userHandler.Create)
	r.Get("/users/{id}", userHandler.GetByID)
	r.Put("/users/{id}", userHandler.Update)
	r.Put("/users/{id}/password", userHandler.UpdatePassword)
	r.Delete("/users/{id}", userHandler.Delete)

	// Composites
	r.Get("/composites", compositeHandler.List)
	r.Post("/composites", compositeHandler.Create)
	r.Get("/composites/slug/{slug}", compositeHandler.GetBySlug)
	r.Get("/composites/{id}", compositeHandler.GetByID)
	r.Put("/composites/{id}", compositeHandler.Update)
	r.Delete("/composites/{id}", compositeHandler.Delete)

	// Fields
	r.Get("/composites/{compositeID}/fields", fieldHandler.ListByComposite)
	r.Post("/fields", fieldHandler.Create)
	r.Get("/fields/{id}", fieldHandler.GetByID)
	r.Put("/fields/{id}", fieldHandler.Update)
	r.Delete("/fields/{id}", fieldHandler.Delete)

	// Entities
	r.Get("/composites/{compositeID}/entities", entityHandler.ListByComposite)
	r.Get("/composites/{compositeID}/entities/slug/{slug}", entityHandler.GetBySlug)
	r.Post("/entities", entityHandler.Create)
	r.Get("/entities/{id}", entityHandler.GetByID)
	r.Put("/entities/{id}", entityHandler.Update)
	r.Delete("/entities/{id}", entityHandler.Delete)
	r.Get("/entities/{id}/children", entityHandler.ListChildren)
	r.Post("/entities/{id}/move", entityHandler.Move)
	r.Post("/entities/{id}/move-root", entityHandler.MoveRoot)

	// Field values
	r.Get("/entities/{entityID}/field-values", fieldValueHandler.ListByEntity)
	r.Post("/field-values", fieldValueHandler.Set)
	r.Get("/field-values/{id}", fieldValueHandler.GetByID)
	r.Delete("/field-values/{id}", fieldValueHandler.Delete)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}
