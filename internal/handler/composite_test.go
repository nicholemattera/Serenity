package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nicholemattera/serenity/internal/models"
)

// setupCompositeAccess creates a role with composite read+write resource permissions
// and a user with that role, returning a valid JWT token.
func setupCompositeAccess(t *testing.T, srv *testServer) string {
	t.Helper()
	ctx := context.Background()

	role, err := srv.roleSvc.Create(ctx, &models.Role{
		Name:           "editor",
		HierarchyLevel: 5,
		SessionTimeout: 3600,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	_, err = srv.permissionSvc.Create(ctx, &models.Permission{
		RoleID:       role.ID,
		ResourceType: ptr(models.ResourceTypeComposite),
		CanRead:      true,
		CanWrite:     true,
	})
	if err != nil {
		t.Fatalf("create composite permission: %v", err)
	}

	_, err = srv.permissionSvc.Create(ctx, &models.Permission{
		RoleID:       role.ID,
		ResourceType: ptr(models.ResourceTypeField),
		CanRead:      true,
		CanWrite:     true,
	})
	if err != nil {
		t.Fatalf("create field permission: %v", err)
	}

	user := &models.User{
		FirstName: "Test",
		LastName:  "Editor",
		Email:     "editor@example.com",
		RoleID:    role.ID,
	}
	_, err = srv.userSvc.Create(ctx, user, "Super-secret_1234")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	token, err := srv.authSvc.Login(ctx, "editor@example.com", "Super-secret_1234")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	return token
}

func ptr[T any](v T) *T { return &v }

func TestComposite_CRUD(t *testing.T) {
	srv := newTestServer(t)
	token := setupCompositeAccess(t, srv)

	var compositeID string

	t.Run("create composite", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/composites", map[string]any{
			"name":          "Blog Posts",
			"slug":          "blog-posts",
			"default_read":  true,
			"default_write": false,
		}, token)
		assertStatus(t, rr, http.StatusCreated)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["slug"] != "blog-posts" {
			t.Errorf("expected slug blog-posts, got %v", resp["slug"])
		}
		compositeID = resp["id"].(string)
	})

	t.Run("get composite by id", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites/"+compositeID, nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["id"] != compositeID {
			t.Errorf("expected id %s, got %v", compositeID, resp["id"])
		}
	})

	t.Run("get composite by slug", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites/slug/blog-posts", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["slug"] != "blog-posts" {
			t.Errorf("expected slug blog-posts, got %v", resp["slug"])
		}
	})

	t.Run("list composites", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["total"].(float64) < 1 {
			t.Error("expected at least one composite in list")
		}
	})

	t.Run("update composite", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/composites/"+compositeID, map[string]any{
			"name":          "Blog Posts Updated",
			"slug":          "blog-posts",
			"default_read":  true,
			"default_write": false,
		}, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["name"] != "Blog Posts Updated" {
			t.Errorf("expected updated name, got %v", resp["name"])
		}
	})

	t.Run("create field on composite", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/fields", map[string]any{
			"composite_id": compositeID,
			"name":         "Title",
			"slug":         "title",
			"type":         "short_text",
			"required":     true,
			"position":     1,
		}, token)
		assertStatus(t, rr, http.StatusCreated)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["slug"] != "title" {
			t.Errorf("expected slug title, got %v", resp["slug"])
		}
	})

	t.Run("get composite enriched includes fields", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites/"+compositeID+"?enrich=true", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		fields := resp["fields"].([]any)
		if len(fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(fields))
		}
	})

	t.Run("delete composite", func(t *testing.T) {
		rr := srv.do(http.MethodDelete, "/v1/composites/"+compositeID, nil, token)
		assertStatus(t, rr, http.StatusNoContent)
	})

	t.Run("get deleted composite returns 404", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites/"+compositeID, nil, token)
		assertStatus(t, rr, http.StatusNotFound)
	})
}

func TestComposite_PermissionEnforcement(t *testing.T) {
	srv := newTestServer(t)
	token := setupCompositeAccess(t, srv)

	// Create a composite to test against.
	rr := srv.do(http.MethodPost, "/v1/composites", map[string]any{
		"name":          "Private",
		"slug":          "private",
		"default_read":  false,
		"default_write": false,
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	var created map[string]any
	decode(t, rr, &created)
	compositeID := created["id"].(string)

	t.Run("unauthenticated cannot create composite", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/composites", map[string]any{
			"name": "x", "slug": "x",
		}, "")
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("unauthenticated cannot list composites", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites", nil, "")
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("unauthenticated cannot get composite by id", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/v1/composites/"+compositeID, nil, "")
		assertStatus(t, rr, http.StatusForbidden)
	})
}
