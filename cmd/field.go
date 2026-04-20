package cmd

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/nicholemattera/serenity/internal/models"
)

func newFieldCmd() *cobra.Command {
	var compositeID string

	cmd := &cobra.Command{
		Use:   "field",
		Short: "Manage fields",
		RunE: func(cmd *cobra.Command, args []string) error {
			if compositeID == "" {
				return fmt.Errorf("--composite-id is required to list fields")
			}
			cid, err := uuid.Parse(compositeID)
			if err != nil {
				return fmt.Errorf("invalid composite-id: %w", err)
			}
			a, err := initApp()
			if err != nil {
				return err
			}
			page, err := a.fieldSvc.ListByComposite(cmd.Context(), cid, nil)
			if err != nil {
				return err
			}
			printJSON(page)
			return nil
		},
	}

	cmd.Flags().StringVar(&compositeID, "composite-id", "", "Composite UUID (required)")

	cmd.AddCommand(
		newFieldCreateCmd(),
		newFieldGetCmd(),
		newFieldUpdateCmd(),
		newFieldDeleteCmd(),
	)

	return cmd
}

func newFieldCreateCmd() *cobra.Command {
	var (
		compositeID  string
		name         string
		slug         string
		fieldType    string
		required     bool
		position     int
		defaultValue string
		hasDefault   bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a field",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid, err := uuid.Parse(compositeID)
			if err != nil {
				return fmt.Errorf("invalid composite-id: %w", err)
			}
			a, err := initApp()
			if err != nil {
				return err
			}

			field := &models.Field{
				CompositeID: cid,
				Name:        name,
				Slug:        slug,
				Type:        models.FieldType(fieldType),
				Required:    required,
				Position:    position,
			}
			if hasDefault {
				field.DefaultValue = &defaultValue
			}

			result, err := a.fieldSvc.Create(cmd.Context(), field)
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&compositeID, "composite-id", "", "Composite UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Field name (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe slug (required)")
	cmd.Flags().StringVar(&fieldType, "type", "", "Field type: association, checkbox, color, date, datetime, dropdown, email, file, long_text, number, phone, short_text, time, url (required)")
	cmd.Flags().BoolVar(&required, "required", false, "Mark field as required")
	cmd.Flags().IntVar(&position, "position", 1, "Display position within the composite")
	cmd.Flags().StringVar(&defaultValue, "default-value", "", "Default value for this field")
	cmd.Flags().BoolVar(&hasDefault, "has-default", false, "Set a default value (use with --default-value)")
	_ = cmd.MarkFlagRequired("composite-id")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("slug")
	_ = cmd.MarkFlagRequired("type")

	return cmd
}

func newFieldGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a field by ID",
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
			field, err := a.fieldSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			printJSON(field)
			return nil
		},
	}
}

func newFieldUpdateCmd() *cobra.Command {
	var (
		name         string
		slug         string
		fieldType    string
		required     bool
		position     int
		defaultValue string
		clearDefault bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a field",
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

			existing, err := a.fieldSvc.GetByID(cmd.Context(), id)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("name") {
				name = existing.Name
			}
			if !cmd.Flags().Changed("slug") {
				slug = existing.Slug
			}
			if !cmd.Flags().Changed("type") {
				fieldType = string(existing.Type)
			}
			if !cmd.Flags().Changed("required") {
				required = existing.Required
			}
			if !cmd.Flags().Changed("position") {
				position = existing.Position
			}

			dv := existing.DefaultValue
			if cmd.Flags().Changed("default-value") {
				dv = &defaultValue
			}
			if clearDefault {
				dv = nil
			}

			result, err := a.fieldSvc.Update(cmd.Context(), &models.Field{
				ID:           id,
				CompositeID:  existing.CompositeID,
				Name:         name,
				Slug:         slug,
				Type:         models.FieldType(fieldType),
				Required:     required,
				Position:     position,
				DefaultValue: dv,
				Metadata:     existing.Metadata,
			})
			if err != nil {
				return err
			}
			printJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Field name")
	cmd.Flags().StringVar(&slug, "slug", "", "URL-safe slug")
	cmd.Flags().StringVar(&fieldType, "type", "", "Field type")
	cmd.Flags().BoolVar(&required, "required", false, "Mark field as required")
	cmd.Flags().IntVar(&position, "position", 0, "Display position")
	cmd.Flags().StringVar(&defaultValue, "default-value", "", "Default value")
	cmd.Flags().BoolVar(&clearDefault, "clear-default", false, "Remove the default value")

	return cmd
}

func newFieldDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a field",
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
			if err := a.fieldSvc.Delete(cmd.Context(), id, uuid.Nil); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
}
