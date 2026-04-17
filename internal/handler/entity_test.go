package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nicholemattera/serenity/internal/models"
)

// setupEntityAccess creates a role, user, and composite configured for entity read+write.
// Returns the token and composite ID.
func setupEntityAccess(t *testing.T, srv *testServer, defaultRead, defaultWrite bool) (token string, compositeID string) {
	t.Helper()
	ctx := context.Background()

	role, err := srv.roleSvc.Create(ctx, &models.Role{
		Name:           "entity-editor",
		HierarchyLevel: 5,
		SessionTimeout: 3600,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	// Grant composite resource read so the user can look up composites.
	for _, rt := range []models.ResourceType{models.ResourceTypeComposite, models.ResourceTypeField} {
		_, err = srv.permissionSvc.Create(ctx, &models.Permission{
			RoleID:       role.ID,
			ResourceType: ptr(rt),
			CanRead:      true,
			CanWrite:     true,
		})
		if err != nil {
			t.Fatalf("create resource permission: %v", err)
		}
	}

	user := &models.User{
		FirstName: "Entity",
		LastName:  "Editor",
		Email:     "entity-editor@example.com",
		RoleID:    role.ID,
	}
	_, err = srv.userSvc.Create(ctx, user, "password123")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	token, err = srv.authSvc.Login(ctx, "entity-editor@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	// Create a composite with the given default access flags.
	rr := srv.do(http.MethodPost, "/composites", map[string]any{
		"name":          "Posts",
		"slug":          "posts",
		"default_read":  defaultRead,
		"default_write": defaultWrite,
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	var composite map[string]any
	decode(t, rr, &composite)
	compositeID = composite["id"].(string)

	return token, compositeID
}

func TestEntity_CRUD(t *testing.T) {
	srv := newTestServer(t)
	token, compositeID := setupEntityAccess(t, srv, true, true)

	var entityID string

	t.Run("create entity", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/entities", map[string]any{
			"composite_id": compositeID,
			"name":         "Hello World",
			"slug":         "hello-world",
		}, token)
		assertStatus(t, rr, http.StatusCreated)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["slug"] != "hello-world" {
			t.Errorf("expected slug hello-world, got %v", resp["slug"])
		}
		entityID = resp["id"].(string)
	})

	t.Run("get entity by id", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/entities/"+entityID, nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["id"] != entityID {
			t.Errorf("expected id %s, got %v", entityID, resp["id"])
		}
	})

	t.Run("get entity by slug", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/composites/"+compositeID+"/entities/slug/hello-world", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["slug"] != "hello-world" {
			t.Errorf("expected slug hello-world, got %v", resp["slug"])
		}
	})

	t.Run("list entities by composite", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/composites/"+compositeID+"/entities", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["total"].(float64) < 1 {
			t.Error("expected at least one entity")
		}
	})

	t.Run("update entity", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/entities/"+entityID, map[string]any{
			"name": "Hello World Updated",
			"slug": "hello-world",
		}, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		if resp["name"] != "Hello World Updated" {
			t.Errorf("expected updated name, got %v", resp["name"])
		}
	})

	t.Run("set field value", func(t *testing.T) {
		// Create a field first.
		fieldRR := srv.do(http.MethodPost, "/fields", map[string]any{
			"composite_id": compositeID,
			"name":         "Body",
			"slug":         "body",
			"type":         "long_text",
			"position":     1,
		}, token)
		assertStatus(t, fieldRR, http.StatusCreated)
		var field map[string]any
		decode(t, fieldRR, &field)

		rr := srv.do(http.MethodPost, "/field-values", map[string]any{
			"entity_id": entityID,
			"field_id":  field["id"],
			"value":     "My first post content",
		}, token)
		assertStatus(t, rr, http.StatusOK)

		var fv map[string]any
		decode(t, rr, &fv)
		if fv["value"] != "My first post content" {
			t.Errorf("expected value, got %v", fv["value"])
		}
	})

	t.Run("get entity enriched includes field values", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/entities/"+entityID+"?enrich=true", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		fvs := resp["field_values"].([]any)
		if len(fvs) != 1 {
			t.Errorf("expected 1 field value, got %d", len(fvs))
		}
	})

	t.Run("delete entity", func(t *testing.T) {
		rr := srv.do(http.MethodDelete, "/entities/"+entityID, nil, token)
		assertStatus(t, rr, http.StatusNoContent)
	})

	t.Run("get deleted entity returns 404", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/entities/"+entityID, nil, token)
		assertStatus(t, rr, http.StatusNotFound)
	})
}

func TestEntity_DefaultReadWrite(t *testing.T) {
	srv := newTestServer(t)
	_, compositeID := setupEntityAccess(t, srv, true, true)

	t.Run("unauthenticated can read entity when default_read=true", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/composites/"+compositeID+"/entities", nil, "")
		assertStatus(t, rr, http.StatusOK)
	})

	t.Run("unauthenticated can create entity when default_write=true", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/entities", map[string]any{
			"composite_id": compositeID,
			"name":         "Anonymous Post",
			"slug":         "anonymous-post",
		}, "")
		assertStatus(t, rr, http.StatusCreated)
	})

	srv2 := newTestServer(t)
	_, closedCompositeID := setupEntityAccess(t, srv2, false, false)

	t.Run("unauthenticated cannot read entity when default_read=false", func(t *testing.T) {
		rr := srv2.do(http.MethodGet, "/composites/"+closedCompositeID+"/entities", nil, "")
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("unauthenticated cannot create entity when default_write=false", func(t *testing.T) {
		rr := srv2.do(http.MethodPost, "/entities", map[string]any{
			"composite_id": closedCompositeID,
			"name":         "Blocked Post",
			"slug":         "blocked-post",
		}, "")
		assertStatus(t, rr, http.StatusForbidden)
	})
}

func TestEntity_ParentChild(t *testing.T) {
	srv := newTestServer(t)
	token, compositeID := setupEntityAccess(t, srv, true, true)

	// Create parent entity.
	rr := srv.do(http.MethodPost, "/entities", map[string]any{
		"composite_id": compositeID,
		"name":         "Parent",
		"slug":         "parent",
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	var parent map[string]any
	decode(t, rr, &parent)
	parentID := parent["id"].(string)

	// Create child entity.
	rr = srv.do(http.MethodPost, "/entities", map[string]any{
		"composite_id": compositeID,
		"name":         "Child",
		"slug":         "child",
		"parent_id":    parentID,
	}, token)
	assertStatus(t, rr, http.StatusCreated)
	var child map[string]any
	decode(t, rr, &child)
	childID := child["id"].(string)

	t.Run("list children returns child", func(t *testing.T) {
		rr := srv.do(http.MethodGet, "/entities/"+parentID+"/children", nil, token)
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]any
		decode(t, rr, &resp)
		children := resp["data"].([]any)
		if len(children) != 1 {
			t.Fatalf("expected 1 child, got %d", len(children))
		}
		c := children[0].(map[string]any)
		if c["id"] != childID {
			t.Errorf("expected child id %s, got %v", childID, c["id"])
		}
	})
}
