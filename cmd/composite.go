package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

func newCompositeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "composite",
		Short: "Manage composites",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}
			page, err := a.compositeSvc.List(cmd.Context(), repository.Pagination{Limit: 100}, false)
			if err != nil {
				return err
			}
			printJSON(page)
			return nil
		},
	}

	cmd.AddCommand(
		newCompositeCreateCmd(),
		newCompositeGetCmd(),
		newCompositeUpdateCmd(),
		newCompositeDeleteCmd(),
	)

	return cmd
}

func newCompositeCreateCmd() *cobra.Command {
	var (
		name         string
		slug         string
		defaultRead  bool
		defaultWrite bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a composite",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := initApp()
			if err != nil {
				return err
			}
			result, err := a.compositeSvc.Create(cmd.Context(), &models.Composite{
				Name:         name,
				Slug:         slug,
				DefaultRead:  defaultRead,
				DefaultWrite: defaultWrite,
			})
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Composite name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe slug (required)")
	cmd.Flags().BoolVar(&defaultRead, "default-read", false, "Allow unauthenticated reads")
	cmd.Flags().BoolVar(&defaultWrite, "default-write", false, "Allow unauthenticated writes")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("slug")

	return cmd
}

func newCompositeGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a composite by ID",
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
			result, err := a.compositeSvc.GetByID(cmd.Context(), id, true)
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}
}

func newCompositeUpdateCmd() *cobra.Command {
	var (
		name         string
		slug         string
		defaultRead  bool
		defaultWrite bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a composite",
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

			existing, err := a.compositeSvc.GetByID(cmd.Context(), id, false)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("name") {
				name = existing.Name
			}
			if !cmd.Flags().Changed("slug") {
				slug = existing.Slug
			}
			if !cmd.Flags().Changed("default-read") {
				defaultRead = existing.DefaultRead
			}
			if !cmd.Flags().Changed("default-write") {
				defaultWrite = existing.DefaultWrite
			}

			result, err := a.compositeSvc.Update(cmd.Context(), &models.Composite{
				ID:           id,
				Name:         name,
				Slug:         slug,
				DefaultRead:  defaultRead,
				DefaultWrite: defaultWrite,
			}, false)
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Composite name")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe slug")
	cmd.Flags().BoolVar(&defaultRead, "default-read", false, "Allow unauthenticated reads")
	cmd.Flags().BoolVar(&defaultWrite, "default-write", false, "Allow unauthenticated writes")

	return cmd
}

func newCompositeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a composite",
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
			if err := a.compositeSvc.Delete(cmd.Context(), id, uuid.Nil); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
}
