package service_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

// --- UserRepository mock ---

type mockUserRepo struct {
	getByEmail func(ctx context.Context, email string) (*models.User, error)
	getByID    func(ctx context.Context, id uuid.UUID) (*models.User, error)
	create     func(ctx context.Context, user *models.User) (*models.User, error)
	update     func(ctx context.Context, user *models.User) (*models.User, error)
}

func (m *mockUserRepo) Create(ctx context.Context, user *models.User) (*models.User, error) {
	return m.create(ctx, user)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, pgx.ErrNoRows
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return m.getByEmail(ctx, email)
}
func (m *mockUserRepo) List(ctx context.Context, p repository.Pagination) (*repository.Page[models.User], error) {
	return nil, nil
}
func (m *mockUserRepo) Update(ctx context.Context, user *models.User) (*models.User, error) {
	if m.update != nil {
		return m.update(ctx, user)
	}
	return user, nil
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	return nil
}

// --- RoleRepository mock ---

type mockRoleRepo struct {
	getByID func(ctx context.Context, id uuid.UUID) (*models.Role, error)
}

func (m *mockRoleRepo) Create(ctx context.Context, role *models.Role) (*models.Role, error) {
	return nil, nil
}
func (m *mockRoleRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	return m.getByID(ctx, id)
}
func (m *mockRoleRepo) List(ctx context.Context, p repository.Pagination) (*repository.Page[models.Role], error) {
	return nil, nil
}
func (m *mockRoleRepo) Update(ctx context.Context, role *models.Role) (*models.Role, error) {
	return nil, nil
}
func (m *mockRoleRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	return nil
}

// --- PermissionRepository mock ---

type mockPermissionRepo struct {
	getByRoleAndComposite func(ctx context.Context, roleID, compositeID uuid.UUID) (*models.Permission, error)
	getByRoleAndResource  func(ctx context.Context, roleID uuid.UUID, resourceType models.ResourceType) (*models.Permission, error)
}

func (m *mockPermissionRepo) Create(ctx context.Context, p *models.Permission) (*models.Permission, error) {
	return nil, nil
}
func (m *mockPermissionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	return nil, nil
}
func (m *mockPermissionRepo) GetByRoleAndComposite(ctx context.Context, roleID, compositeID uuid.UUID) (*models.Permission, error) {
	return m.getByRoleAndComposite(ctx, roleID, compositeID)
}
func (m *mockPermissionRepo) GetByRoleAndResource(ctx context.Context, roleID uuid.UUID, resourceType models.ResourceType) (*models.Permission, error) {
	if m.getByRoleAndResource != nil {
		return m.getByRoleAndResource(ctx, roleID, resourceType)
	}
	return nil, pgx.ErrNoRows
}
func (m *mockPermissionRepo) ListByRole(ctx context.Context, roleID uuid.UUID, p repository.Pagination) (*repository.Page[models.Permission], error) {
	return nil, nil
}
func (m *mockPermissionRepo) Update(ctx context.Context, p *models.Permission) (*models.Permission, error) {
	return nil, nil
}
func (m *mockPermissionRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	return nil
}

// --- FieldValueRepository mock ---

type mockFieldValueRepo struct {
	getByEntityAndField func(ctx context.Context, entityID, fieldID uuid.UUID) (*models.FieldValue, error)
	create              func(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error)
	update              func(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error)
}

func (m *mockFieldValueRepo) Create(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
	return m.create(ctx, fv)
}
func (m *mockFieldValueRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.FieldValue, error) {
	return nil, pgx.ErrNoRows
}
func (m *mockFieldValueRepo) GetByEntityAndField(ctx context.Context, entityID, fieldID uuid.UUID) (*models.FieldValue, error) {
	return m.getByEntityAndField(ctx, entityID, fieldID)
}
func (m *mockFieldValueRepo) ListByEntity(ctx context.Context, entityID uuid.UUID, p repository.Pagination) (*repository.Page[models.FieldValue], error) {
	return &repository.Page[models.FieldValue]{Data: []models.FieldValue{}}, nil
}
func (m *mockFieldValueRepo) Update(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
	return m.update(ctx, fv)
}
func (m *mockFieldValueRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	return nil
}
