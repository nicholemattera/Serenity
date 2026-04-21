package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/config"
	"github.com/nicholemattera/serenity/internal/database"
	"github.com/nicholemattera/serenity/internal/repository"
	"github.com/nicholemattera/serenity/internal/service"
)

type appState struct {
	cfg           *config.Config
	db            *pgxpool.Pool
	roleSvc       service.RoleService
	userSvc       service.UserService
	permissionSvc service.PermissionService
	authSvc       service.AuthService
	fieldSvc      service.FieldService
	compositeSvc  service.CompositeService
	fieldValueSvc service.FieldValueService
	entitySvc     service.EntityService
}

var state *appState

func initApp() (*appState, error) {
	if state != nil {
		return state, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)
	permissionRepo := repository.NewPermissionRepository(db)
	compositeRepo := repository.NewCompositeRepository(db)
	fieldRepo := repository.NewFieldRepository(db)
	entityRepo := repository.NewEntityRepository(db)
	fieldValueRepo := repository.NewFieldValueRepository(db)

	roleSvc := service.NewRoleService(roleRepo)
	userSvc := service.NewUserService(userRepo, cfg.BCryptCost)
	permissionSvc := service.NewPermissionService(permissionRepo, cfg.PermissionCacheTTL, cfg.PermissionCacheMaxSize)
	authSvc := service.NewAuthService(userRepo, roleRepo, cfg.JWTSecret)
	fieldSvc := service.NewFieldService(fieldRepo)
	compositeSvc := service.NewCompositeService(compositeRepo, fieldSvc)
	fieldValueSvc := service.NewFieldValueService(fieldValueRepo, fieldSvc)
	entitySvc := service.NewEntityService(entityRepo, fieldValueSvc)

	state = &appState{
		cfg:           cfg,
		db:            db,
		roleSvc:       roleSvc,
		userSvc:       userSvc,
		permissionSvc: permissionSvc,
		authSvc:       authSvc,
		fieldSvc:      fieldSvc,
		compositeSvc:  compositeSvc,
		fieldValueSvc: fieldValueSvc,
		entitySvc:     entitySvc,
	}

	return state, nil
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serenity",
		Short: "Serenity CMS — CLI and service runner",
	}

	cmd.AddCommand(
		newServiceCmd(),
		newRoleCmd(),
		newUserCmd(),
		newPermissionCmd(),
		newCompositeCmd(),
		newFieldCmd(),
	)

	return cmd
}
