package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nicholemattera/serenity/internal/database"
	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

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

func seedField(t *testing.T, repo repository.FieldRepository, compositeID uuid.UUID) *models.Field {
	t.Helper()
	f, err := repo.Create(context.Background(), &models.Field{
		CompositeID: compositeID,
		Name:        "Test Field",
		Slug:        uuid.NewString(),
		Type:        models.FieldTypeShortText,
	})
	if err != nil {
		t.Fatalf("seedField: %v", err)
	}
	return f
}

func seedEntity(t *testing.T, repo repository.EntityRepository, compositeID uuid.UUID) *models.Entity {
	t.Helper()
	e, err := repo.Create(context.Background(), &models.Entity{
		CompositeID: compositeID,
		Name:        uuid.NewString(),
		Slug:        uuid.NewString(),
	}, nil, nil)
	if err != nil {
		t.Fatalf("seedEntity: %v", err)
	}
	return e
}
