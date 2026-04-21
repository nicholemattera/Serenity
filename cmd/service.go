package cmd

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
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/handler"
)

func newServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "service",
		Short: "Start the HTTP service",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}

			authHandler := handler.NewAuthHandler(a.authSvc, a.userSvc, a.roleSvc)
			roleHandler := handler.NewRoleHandler(a.roleSvc, a.permissionSvc)
			permissionHandler := handler.NewPermissionHandler(a.permissionSvc)
			userHandler := handler.NewUserHandler(a.userSvc, a.permissionSvc)
			compositeHandler := handler.NewCompositeHandler(a.compositeSvc, a.permissionSvc)
			fieldHandler := handler.NewFieldHandler(a.fieldSvc, a.permissionSvc)
			entityHandler := handler.NewEntityHandler(a.entitySvc, a.fieldSvc, a.fieldValueSvc, a.compositeSvc, a.permissionSvc)
			fieldValueHandler := handler.NewFieldValueHandler(a.fieldValueSvc, a.entitySvc, a.compositeSvc, a.permissionSvc)

			r := chi.NewRouter()
			r.Use(middleware.Logger)
			r.Use(middleware.Recoverer)
			r.Use(middleware.RequestID)
			r.Use(handler.Authenticate(a.authSvc))

			r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})

			r.With(handler.RateLimit(5, time.Minute)).Post("/auth/login", authHandler.Login)
			r.With(handler.RateLimit(3, time.Minute)).Post("/auth/register", authHandler.Register)

			r.Get("/roles", roleHandler.List)
			r.Post("/roles", roleHandler.Create)
			r.Get("/roles/{id}", roleHandler.GetByID)
			r.Put("/roles/{id}", roleHandler.Update)
			r.Delete("/roles/{id}", roleHandler.Delete)
			r.Get("/roles/{roleID}/permissions", permissionHandler.ListByRole)

			r.Post("/permissions", permissionHandler.Create)
			r.Get("/permissions/{id}", permissionHandler.GetByID)
			r.Put("/permissions/{id}", permissionHandler.Update)
			r.Delete("/permissions/{id}", permissionHandler.Delete)

			r.Get("/users", userHandler.List)
			r.Post("/users", userHandler.Create)
			r.Get("/users/{id}", userHandler.GetByID)
			r.Put("/users/{id}", userHandler.Update)
			r.Put("/users/{id}/password", userHandler.UpdatePassword)
			r.Delete("/users/{id}", userHandler.Delete)

			r.Get("/composites", compositeHandler.List)
			r.Post("/composites", compositeHandler.Create)
			r.Get("/composites/slug/{slug}", compositeHandler.GetBySlug)
			r.Get("/composites/{id}", compositeHandler.GetByID)
			r.Put("/composites/{id}", compositeHandler.Update)
			r.Delete("/composites/{id}", compositeHandler.Delete)

			r.Get("/composites/{compositeID}/fields", fieldHandler.ListByComposite)
			r.Post("/fields", fieldHandler.Create)
			r.Get("/fields/{id}", fieldHandler.GetByID)
			r.Put("/fields/{id}", fieldHandler.Update)
			r.Delete("/fields/{id}", fieldHandler.Delete)

			r.Get("/composites/{compositeID}/entities", entityHandler.ListByComposite)
			r.Get("/composites/{compositeID}/entities/slug/{slug}", entityHandler.GetBySlug)
			r.Post("/entities", entityHandler.Create)
			r.Get("/entities/{id}", entityHandler.GetByID)
			r.Put("/entities/{id}", entityHandler.Update)
			r.Delete("/entities/{id}", entityHandler.Delete)
			r.Get("/entities/{id}/children", entityHandler.ListChildren)
			r.Post("/entities/{id}/move", entityHandler.Move)
			r.Post("/entities/{id}/move-root", entityHandler.MoveRoot)

			r.Get("/entities/{entityID}/field-values", fieldValueHandler.ListByEntity)
			r.Post("/field-values", fieldValueHandler.Set)
			r.Get("/field-values/{id}", fieldValueHandler.GetByID)
			r.Delete("/field-values/{id}", fieldValueHandler.Delete)

			srv := &http.Server{
				Addr:    ":" + a.cfg.Port,
				Handler: r,
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				slog.Info("server listening", "port", a.cfg.Port)
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
				return err
			}

			slog.Info("server stopped")
			return nil
		},
	}
}
