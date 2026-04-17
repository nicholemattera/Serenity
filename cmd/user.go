package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}
			page, err := a.userSvc.List(cmd.Context(), repository.Pagination{Limit: 100})
			if err != nil {
				return err
			}
			printJSON(page)
			return nil
		},
	}

	cmd.AddCommand(
		newUserCreateCmd(),
		newUserGetCmd(),
		newUserUpdateCmd(),
		newUserDeleteCmd(),
	)

	return cmd
}

func newUserCreateCmd() *cobra.Command {
	var (
		firstName string
		lastName  string
		email     string
		password  string
		roleID    string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			rid, err := uuid.Parse(roleID)
			if err != nil {
				return fmt.Errorf("invalid role-id: %w", err)
			}
			a, err := initApp()
			if err != nil {
				return err
			}
			user, err := a.userSvc.Create(cmd.Context(), &models.User{
				FirstName: firstName,
				LastName:  lastName,
				Email:     email,
				RoleID:    rid,
			}, password)
			if err != nil {
				return err
			}
			printJSON(user)
			return nil
		},
	}

	cmd.Flags().StringVar(&firstName, "first-name", "", "First name (required)")
	cmd.Flags().StringVar(&lastName, "last-name", "", "Last name (required)")
	cmd.Flags().StringVar(&email, "email", "", "Email address (required)")
	cmd.Flags().StringVar(&password, "password", "", "Password (required)")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Role UUID (required)")
	_ = cmd.MarkFlagRequired("first-name")
	_ = cmd.MarkFlagRequired("last-name")
	_ = cmd.MarkFlagRequired("email")
	_ = cmd.MarkFlagRequired("password")
	_ = cmd.MarkFlagRequired("role-id")

	return cmd
}

func newUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a user by ID",
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
			user, err := a.userSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			printJSON(user)
			return nil
		},
	}
}

func newUserUpdateCmd() *cobra.Command {
	var (
		firstName string
		lastName  string
		email     string
		roleID    string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a user",
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

			existing, err := a.userSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("first-name") {
				firstName = existing.FirstName
			}
			if !cmd.Flags().Changed("last-name") {
				lastName = existing.LastName
			}
			if !cmd.Flags().Changed("email") {
				email = existing.Email
			}
			rid := existing.RoleID
			if cmd.Flags().Changed("role-id") {
				rid, err = uuid.Parse(roleID)
				if err != nil {
					return fmt.Errorf("invalid role-id: %w", err)
				}
			}

			user, err := a.userSvc.Update(cmd.Context(), &models.User{
				ID:        id,
				FirstName: firstName,
				LastName:  lastName,
				Email:     email,
				RoleID:    rid,
			})
			if err != nil {
				return err
			}
			printJSON(user)
			return nil
		},
	}

	cmd.Flags().StringVar(&firstName, "first-name", "", "First name")
	cmd.Flags().StringVar(&lastName, "last-name", "", "Last name")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&roleID, "role-id", "", "Role UUID")

	return cmd
}

func newUserDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a user",
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
			if err := a.userSvc.Delete(cmd.Context(), id, uuid.Nil); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
}
