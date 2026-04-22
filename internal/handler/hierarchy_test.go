package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nicholemattera/serenity/internal/models"
)

func TestUserHierarchy(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	adminRole, err := srv.roleSvc.Create(ctx, &models.Role{Name: "admin", HierarchyLevel: 1, SessionTimeout: 3600})
	if err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	managerRole, err := srv.roleSvc.Create(ctx, &models.Role{Name: "manager", HierarchyLevel: 5, SessionTimeout: 3600})
	if err != nil {
		t.Fatalf("create manager role: %v", err)
	}
	memberRole, err := srv.roleSvc.Create(ctx, &models.Role{Name: "member", HierarchyLevel: 10, SessionTimeout: 3600})
	if err != nil {
		t.Fatalf("create member role: %v", err)
	}

	_, err = srv.permissionSvc.Create(ctx, &models.Permission{
		RoleID:       managerRole.ID,
		ResourceType: ptr(models.ResourceTypeUser),
		CanRead:      true,
		CanWrite:     true,
	})
	if err != nil {
		t.Fatalf("create user permission for manager: %v", err)
	}

	_, err = srv.userSvc.Create(ctx, &models.User{
		FirstName: "Mgr", LastName: "User", Email: "mgr@example.com", RoleID: managerRole.ID,
	}, "Super-secret_1234")
	if err != nil {
		t.Fatalf("create manager user: %v", err)
	}
	managerToken, err := srv.authSvc.Login(ctx, "mgr@example.com", "Super-secret_1234")
	if err != nil {
		t.Fatalf("login as manager: %v", err)
	}

	memberUser, err := srv.userSvc.Create(ctx, &models.User{
		FirstName: "Member", LastName: "User", Email: "member@example.com", RoleID: memberRole.ID,
	}, "Super-secret_1234")
	if err != nil {
		t.Fatalf("create member user: %v", err)
	}

	adminUser, err := srv.userSvc.Create(ctx, &models.User{
		FirstName: "Admin", LastName: "User", Email: "admin@example.com", RoleID: adminRole.ID,
	}, "Super-secret_1234")
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	peerUser, err := srv.userSvc.Create(ctx, &models.User{
		FirstName: "Peer", LastName: "Manager", Email: "peer@example.com", RoleID: managerRole.ID,
	}, "Super-secret_1234")
	if err != nil {
		t.Fatalf("create peer manager user: %v", err)
	}

	t.Run("create: allowed for lower-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/users", map[string]any{
			"first_name": "New", "last_name": "Member", "email": "newmember@example.com",
			"password": "Super-secret_1234", "role_id": memberRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusCreated)
	})

	t.Run("create: blocked for same-level role", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/users", map[string]any{
			"first_name": "New", "last_name": "Manager", "email": "newmanager@example.com",
			"password": "Super-secret_1234", "role_id": managerRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("create: blocked for higher-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/users", map[string]any{
			"first_name": "New", "last_name": "Admin", "email": "newadmin@example.com",
			"password": "Super-secret_1234", "role_id": adminRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: allowed for lower-authority user", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+memberUser.ID.String(), map[string]any{
			"first_name": "Updated", "last_name": "Member",
			"email": "member@example.com", "role_id": memberRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusOK)
	})

	t.Run("update: blocked for higher-authority user", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+adminUser.ID.String(), map[string]any{
			"first_name": "Hacked", "last_name": "Admin",
			"email": "admin@example.com", "role_id": adminRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: blocked for peer user", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+peerUser.ID.String(), map[string]any{
			"first_name": "Hacked", "last_name": "Peer",
			"email": "peer@example.com", "role_id": managerRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: blocked when promoting to same-level role", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+memberUser.ID.String(), map[string]any{
			"first_name": "Member", "last_name": "User",
			"email": "member@example.com", "role_id": managerRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: blocked when promoting to higher-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+memberUser.ID.String(), map[string]any{
			"first_name": "Member", "last_name": "User",
			"email": "member@example.com", "role_id": adminRole.ID,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("delete: allowed for lower-authority user", func(t *testing.T) {
		target, err := srv.userSvc.Create(ctx, &models.User{
			FirstName: "ToDelete", LastName: "User", Email: "todelete@example.com", RoleID: memberRole.ID,
		}, "Super-secret_1234")
		if err != nil {
			t.Fatalf("create user to delete: %v", err)
		}
		rr := srv.do(http.MethodDelete, "/v1/users/"+target.ID.String(), nil, managerToken)
		assertStatus(t, rr, http.StatusNoContent)
	})

	t.Run("delete: blocked for higher-authority user", func(t *testing.T) {
		rr := srv.do(http.MethodDelete, "/v1/users/"+adminUser.ID.String(), nil, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("delete: blocked for peer user", func(t *testing.T) {
		rr := srv.do(http.MethodDelete, "/v1/users/"+peerUser.ID.String(), nil, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update password: allowed for lower-authority user", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+memberUser.ID.String()+"/password", map[string]any{
			"password": "New-password_5678",
		}, managerToken)
		assertStatus(t, rr, http.StatusNoContent)
	})

	t.Run("update password: blocked for higher-authority user", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+adminUser.ID.String()+"/password", map[string]any{
			"password": "New-password_5678",
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update password: blocked for peer user", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/users/"+peerUser.ID.String()+"/password", map[string]any{
			"password": "New-password_5678",
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})
}

func TestRoleHierarchy(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	adminRole, err := srv.roleSvc.Create(ctx, &models.Role{Name: "admin", HierarchyLevel: 1, SessionTimeout: 3600})
	if err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	managerRole, err := srv.roleSvc.Create(ctx, &models.Role{Name: "manager", HierarchyLevel: 5, SessionTimeout: 3600})
	if err != nil {
		t.Fatalf("create manager role: %v", err)
	}
	memberRole, err := srv.roleSvc.Create(ctx, &models.Role{Name: "member", HierarchyLevel: 10, SessionTimeout: 3600})
	if err != nil {
		t.Fatalf("create member role: %v", err)
	}

	_, err = srv.permissionSvc.Create(ctx, &models.Permission{
		RoleID:       managerRole.ID,
		ResourceType: ptr(models.ResourceTypeRole),
		CanRead:      true,
		CanWrite:     true,
	})
	if err != nil {
		t.Fatalf("create role permission for manager: %v", err)
	}

	_, err = srv.userSvc.Create(ctx, &models.User{
		FirstName: "Mgr", LastName: "User", Email: "mgr@example.com", RoleID: managerRole.ID,
	}, "Super-secret_1234")
	if err != nil {
		t.Fatalf("create manager user: %v", err)
	}
	managerToken, err := srv.authSvc.Login(ctx, "mgr@example.com", "Super-secret_1234")
	if err != nil {
		t.Fatalf("login as manager: %v", err)
	}

	t.Run("create: allowed for lower-authority level", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/roles", map[string]any{
			"name": "staff", "hierarchy_level": 8, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusCreated)
	})

	t.Run("create: blocked for same-level", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/roles", map[string]any{
			"name": "peer2", "hierarchy_level": 5, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("create: blocked for higher-authority level", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/roles", map[string]any{
			"name": "superadmin", "hierarchy_level": 1, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: allowed for lower-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/roles/"+memberRole.ID.String(), map[string]any{
			"name": "member-updated", "hierarchy_level": 10, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusOK)
	})

	t.Run("update: blocked for same-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/roles/"+managerRole.ID.String(), map[string]any{
			"name": "manager-hacked", "hierarchy_level": 5, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: blocked for higher-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/roles/"+adminRole.ID.String(), map[string]any{
			"name": "admin-hacked", "hierarchy_level": 1, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: blocked when changing level to same-authority", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/roles/"+memberRole.ID.String(), map[string]any{
			"name": "member", "hierarchy_level": 5, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("update: blocked when changing level to higher-authority", func(t *testing.T) {
		rr := srv.do(http.MethodPut, "/v1/roles/"+memberRole.ID.String(), map[string]any{
			"name": "member", "hierarchy_level": 1, "session_timeout": 3600,
		}, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("delete: allowed for lower-authority role", func(t *testing.T) {
		target, err := srv.roleSvc.Create(ctx, &models.Role{Name: "todelete", HierarchyLevel: 20, SessionTimeout: 3600})
		if err != nil {
			t.Fatalf("create role to delete: %v", err)
		}
		rr := srv.do(http.MethodDelete, "/v1/roles/"+target.ID.String(), nil, managerToken)
		assertStatus(t, rr, http.StatusNoContent)
	})

	t.Run("delete: blocked for same-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodDelete, "/v1/roles/"+managerRole.ID.String(), nil, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})

	t.Run("delete: blocked for higher-authority role", func(t *testing.T) {
		rr := srv.do(http.MethodDelete, "/v1/roles/"+adminRole.ID.String(), nil, managerToken)
		assertStatus(t, rr, http.StatusForbidden)
	})
}
