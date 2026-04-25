package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/models"
)

func newPermissionCmd() *cobra.Command {
	var roleID string

	cmd := &cobra.Command{
		Use:   "permission",
		Short: "Manage permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}
			if roleID == "" {
				return fmt.Errorf("--role-id is required to list permissions")
			}
			rid, err := uuid.Parse(roleID)
			if err != nil {
				return fmt.Errorf("invalid role-id: %w", err)
			}
			page, err := a.permissionSvc.ListByRole(cmd.Context(), rid, nil)
			if err != nil {
				return err
			}
			printJSON(page)
			return nil
		},
	}

	cmd.Flags().StringVar(&roleID, "role-id", "", "Filter permissions by role UUID (required)")

	cmd.AddCommand(
		newPermissionCreateCmd(),
		newPermissionGetCmd(),
		newPermissionUpdateCmd(),
		newPermissionDeleteCmd(),
	)

	return cmd
}

func newPermissionCreateCmd() *cobra.Command {
	var (
		roleID       string
		compositeID  string
		resourceType string
		canRead      bool
		canWrite     bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a permission",
		Long: `Create a permission for a role. Provide either --composite-id (for a user-defined
composite) or --resource-type (for a built-in resource: composite, field, user, role, entity, field_value, permission).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if compositeID == "" && resourceType == "" {
				return fmt.Errorf("one of --composite-id or --resource-type is required")
			}
			if compositeID != "" && resourceType != "" {
				return fmt.Errorf("--composite-id and --resource-type are mutually exclusive")
			}

			rid, err := uuid.Parse(roleID)
			if err != nil {
				return fmt.Errorf("invalid role-id: %w", err)
			}

			p := &models.Permission{
				RoleID:   rid,
				CanRead:  canRead,
				CanWrite: canWrite,
			}

			if compositeID != "" {
				cid, err := uuid.Parse(compositeID)
				if err != nil {
					return fmt.Errorf("invalid composite-id: %w", err)
				}
				p.CompositeID = &cid
			} else {
				rt := models.ResourceType(resourceType)
				p.ResourceType = &rt
			}

			a, err := initApp()
			if err != nil {
				return err
			}
			result, err := a.permissionSvc.Create(cmd.Context(), p)
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&roleID, "role-id", "", "Role UUID (required)")
	cmd.Flags().StringVar(&compositeID, "composite-id", "", "Composite UUID (mutually exclusive with --resource-type)")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "Built-in resource type: composite, field, user, role, entity, field_value, permission (mutually exclusive with --composite-id)")
	cmd.Flags().BoolVar(&canRead, "can-read", false, "Grant read access")
	cmd.Flags().BoolVar(&canWrite, "can-write", false, "Grant write access")
	_ = cmd.MarkFlagRequired("role-id")

	return cmd
}

func newPermissionGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a permission by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			a, err := initApp()
			if err != nil {
				return err
			}
			p, err := a.permissionSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			printJSON(p)
			return nil
		},
	}
}

func newPermissionUpdateCmd() *cobra.Command {
	var (
		canRead  bool
		canWrite bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a permission's can_read / can_write flags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			a, err := initApp()
			if err != nil {
				return err
			}

			existing, err := a.permissionSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("can-read") {
				canRead = existing.CanRead
			}
			if !cmd.Flags().Changed("can-write") {
				canWrite = existing.CanWrite
			}

			existing.CanRead = canRead
			existing.CanWrite = canWrite

			result, err := a.permissionSvc.Update(cmd.Context(), existing)
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&canRead, "can-read", false, "Grant read access")
	cmd.Flags().BoolVar(&canWrite, "can-write", false, "Grant write access")

	return cmd
}

func newPermissionDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a permission",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			a, err := initApp()
			if err != nil {
				return err
			}
			if err := a.permissionSvc.Delete(cmd.Context(), id, uuid.Nil); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
}
