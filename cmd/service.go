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

			trustedProxies, err := handler.ParseTrustedProxies(a.cfg.TrustedProxyIps)
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

			r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})

			r.Route("/v1", func(v1Router chi.Router) {
				v1Router.Use(handler.Authenticate(a.authSvc))

				v1Router.With(handler.RateLimit(a.cfg.LoginRateLimit, a.cfg.LoginRateLimitWindow, trustedProxies)).Post("/auth/login", authHandler.Login)
				v1Router.With(handler.RateLimit(a.cfg.RegisterRateLimit, a.cfg.RegisterRateLimitWindow, trustedProxies)).Post("/auth/register", authHandler.Register)

				v1Router.Get("/roles", roleHandler.List)
				v1Router.With(handler.RequireAuth).Post("/roles", roleHandler.Create)
				v1Router.Get("/roles/{id}", roleHandler.GetByID)
				v1Router.With(handler.RequireAuth).Put("/roles/{id}", roleHandler.Update)
				v1Router.With(handler.RequireAuth).Delete("/roles/{id}", roleHandler.Delete)
				v1Router.Get("/roles/{roleID}/permissions", permissionHandler.ListByRole)

				v1Router.With(handler.RequireAuth).Post("/permissions", permissionHandler.Create)
				v1Router.Get("/permissions/{id}", permissionHandler.GetByID)
				v1Router.With(handler.RequireAuth).Put("/permissions/{id}", permissionHandler.Update)
				v1Router.With(handler.RequireAuth).Delete("/permissions/{id}", permissionHandler.Delete)

				v1Router.Get("/users", userHandler.List)
				v1Router.Post("/users", userHandler.Create)
				v1Router.Get("/users/{id}", userHandler.GetByID)
				v1Router.With(handler.RequireAuth).Put("/users/{id}", userHandler.Update)
				v1Router.With(handler.RateLimit(a.cfg.PasswordUpdateRateLimit, a.cfg.PasswordUpdateRateLimitWindow, trustedProxies)).With(handler.RequireAuth).Put("/users/{id}/password", userHandler.UpdatePassword)
				v1Router.With(handler.RequireAuth).Delete("/users/{id}", userHandler.Delete)

				v1Router.Get("/composites", compositeHandler.List)
				v1Router.With(handler.RequireAuth).Post("/composites", compositeHandler.Create)
				v1Router.Get("/composites/slug/{slug}", compositeHandler.GetBySlug)
				v1Router.Get("/composites/{id}", compositeHandler.GetByID)
				v1Router.With(handler.RequireAuth).Put("/composites/{id}", compositeHandler.Update)
				v1Router.With(handler.RequireAuth).Delete("/composites/{id}", compositeHandler.Delete)

				v1Router.Get("/composites/{compositeID}/fields", fieldHandler.ListByComposite)
				v1Router.With(handler.RequireAuth).Post("/fields", fieldHandler.Create)
				v1Router.Get("/fields/{id}", fieldHandler.GetByID)
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
