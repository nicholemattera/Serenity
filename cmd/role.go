package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/models"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage roles",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}
			page, err := a.roleSvc.List(cmd.Context(), nil)
			if err != nil {
				return err
			}
			printJSON(page)
			return nil
		},
	}

	cmd.AddCommand(
		newRoleCreateCmd(),
		newRoleGetCmd(),
		newRoleUpdateCmd(),
		newRoleDeleteCmd(),
	)

	return cmd
}

func newRoleCreateCmd() *cobra.Command {
	var (
		name              string
		hierarchyLevel    int
		sessionTimeout    int
		allowRegistration bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a role",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}
			role, err := a.roleSvc.Create(cmd.Context(), &models.Role{
				Name:              name,
				HierarchyLevel:    hierarchyLevel,
				SessionTimeout:    sessionTimeout,
				AllowRegistration: allowRegistration,
			})
			if err != nil {
				return err
			}
			printJSON(role)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Role name (required)")
	cmd.Flags().IntVar(&hierarchyLevel, "hierarchy-level", 0, "Hierarchy level — lower numbers indicate higher privilege (required)")
	cmd.Flags().IntVar(&sessionTimeout, "session-timeout", 3600, "JWT session timeout in seconds")
	cmd.Flags().BoolVar(&allowRegistration, "allow-registration", false, "Allow unauthenticated self-registration")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("hierarchy-level")

	return cmd
}

func newRoleGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a role by ID",
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
			role, err := a.roleSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			printJSON(role)
			return nil
		},
	}
}

func newRoleUpdateCmd() *cobra.Command {
	var (
		name              string
		hierarchyLevel    int
		sessionTimeout    int
		allowRegistration bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a role",
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

			// Load current values so unset flags keep their existing value.
			existing, err := a.roleSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("name") {
				name = existing.Name
			}
			if !cmd.Flags().Changed("hierarchy-level") {
				hierarchyLevel = existing.HierarchyLevel
			}
			if !cmd.Flags().Changed("session-timeout") {
				sessionTimeout = existing.SessionTimeout
			}
			if !cmd.Flags().Changed("allow-registration") {
				allowRegistration = existing.AllowRegistration
			}

			role, err := a.roleSvc.Update(cmd.Context(), &models.Role{
				ID:                id,
				Name:              name,
				HierarchyLevel:    hierarchyLevel,
				SessionTimeout:    sessionTimeout,
				AllowRegistration: allowRegistration,
			})
			if err != nil {
				return err
			}
			printJSON(role)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Role name")
	cmd.Flags().IntVar(&hierarchyLevel, "hierarchy-level", 0, "Hierarchy level")
	cmd.Flags().IntVar(&sessionTimeout, "session-timeout", 0, "Session timeout in seconds")
	cmd.Flags().BoolVar(&allowRegistration, "allow-registration", false, "Allow unauthenticated self-registration")

	return cmd
}

func newRoleDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a role",
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
			if err := a.roleSvc.Delete(cmd.Context(), id, uuid.Nil); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
}
