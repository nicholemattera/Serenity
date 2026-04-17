package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

func newFieldValueSvc(fieldType models.FieldType, metadata []byte, fvRepo *mockFieldValueRepo) service.FieldValueService {
	fieldID := uuid.New()
	return service.NewFieldValueService(fvRepo, &mockFieldService{
		getByID: func(_ context.Context, _ uuid.UUID) (*models.Field, error) {
			return &models.Field{ID: fieldID, Type: fieldType, Metadata: metadata}, nil
		},
	})
}

func TestFieldValueService_Set(t *testing.T) {
	ctx := context.Background()
	entityID := uuid.New()
	fieldID := uuid.New()

	existingFV := &models.FieldValue{
		ID:       uuid.New(),
		EntityID: entityID,
		FieldID:  fieldID,
		Value:    "old value",
	}

	t.Run("creates field value when none exists", func(t *testing.T) {
		created := false
		svc := service.NewFieldValueService(&mockFieldValueRepo{
			getByEntityAndField: func(_ context.Context, _, _ uuid.UUID) (*models.FieldValue, error) {
				return nil, pgx.ErrNoRows
			},
			create: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				created = true
				return fv, nil
			},
			update: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				t.Error("update should not be called when no existing value")
				return nil, nil
			},
		}, &mockFieldService{})

		_, err := svc.Set(ctx, &models.FieldValue{EntityID: entityID, FieldID: fieldID, Value: "new value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Error("expected Create to be called")
		}
	})

	t.Run("updates field value when one already exists", func(t *testing.T) {
		updated := false
		var updatedValue string
		svc := service.NewFieldValueService(&mockFieldValueRepo{
			getByEntityAndField: func(_ context.Context, _, _ uuid.UUID) (*models.FieldValue, error) {
				return existingFV, nil
			},
			create: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				t.Error("create should not be called when existing value present")
				return nil, nil
			},
			update: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				updated = true
				updatedValue = fv.Value
				return fv, nil
			},
		}, &mockFieldService{})

		_, err := svc.Set(ctx, &models.FieldValue{EntityID: entityID, FieldID: fieldID, Value: "updated value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated {
			t.Error("expected Update to be called")
		}
		if updatedValue != "updated value" {
			t.Errorf("expected updated value %q, got %q", "updated value", updatedValue)
		}
	})
}

func TestFieldValueService_Validate(t *testing.T) {
	ctx := context.Background()
	entityID := uuid.New()
	fieldID := uuid.New()

	noopRepo := &mockFieldValueRepo{
		getByEntityAndField: func(_ context.Context, _, _ uuid.UUID) (*models.FieldValue, error) {
			return nil, pgx.ErrNoRows
		},
		create: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) { return fv, nil },
		update: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) { return fv, nil },
	}

	set := func(svc service.FieldValueService, value string) error {
		_, err := svc.Set(ctx, &models.FieldValue{EntityID: entityID, FieldID: fieldID, Value: value})
		return err
	}

	tests := []struct {
		name      string
		fieldType models.FieldType
		metadata  []byte
		value     string
		wantErr   bool
	}{
		// checkbox
		{name: "checkbox true", fieldType: models.FieldTypeCheckbox, value: "true"},
		{name: "checkbox false", fieldType: models.FieldTypeCheckbox, value: "false"},
		{name: "checkbox invalid", fieldType: models.FieldTypeCheckbox, value: "yes", wantErr: true},

		// number
		{name: "number integer", fieldType: models.FieldTypeNumber, value: "42"},
		{name: "number float", fieldType: models.FieldTypeNumber, value: "3.14"},
		{name: "number negative", fieldType: models.FieldTypeNumber, value: "-7"},
		{name: "number invalid", fieldType: models.FieldTypeNumber, value: "abc", wantErr: true},

		// email
		{name: "email valid", fieldType: models.FieldTypeEmail, value: "user@example.com"},
		{name: "email invalid", fieldType: models.FieldTypeEmail, value: "not-an-email", wantErr: true},

		// url
		{name: "url valid", fieldType: models.FieldTypeURL, value: "https://example.com"},
		{name: "url no scheme", fieldType: models.FieldTypeURL, value: "example.com", wantErr: true},

		// phone
		{name: "phone valid", fieldType: models.FieldTypePhone, value: "+1 (555) 123-4567"},
		{name: "phone invalid", fieldType: models.FieldTypePhone, value: "abc", wantErr: true},

		// color
		{name: "color #RRGGBB", fieldType: models.FieldTypeColor, value: "#ff5733"},
		{name: "color #RGB", fieldType: models.FieldTypeColor, value: "#f53"},
		{name: "color invalid", fieldType: models.FieldTypeColor, value: "red", wantErr: true},

		// date
		{name: "date valid", fieldType: models.FieldTypeDate, value: "2024-03-15"},
		{name: "date invalid", fieldType: models.FieldTypeDate, value: "15/03/2024", wantErr: true},

		// datetime
		{name: "datetime valid", fieldType: models.FieldTypeDateTime, value: "2024-03-15T14:30:00Z"},
		{name: "datetime invalid", fieldType: models.FieldTypeDateTime, value: "2024-03-15 14:30:00", wantErr: true},

		// time
		{name: "time HH:MM", fieldType: models.FieldTypeTime, value: "14:30"},
		{name: "time HH:MM:SS", fieldType: models.FieldTypeTime, value: "14:30:00"},
		{name: "time invalid", fieldType: models.FieldTypeTime, value: "2pm", wantErr: true},

		// dropdown
		{name: "dropdown valid", fieldType: models.FieldTypeDropdown, metadata: []byte(`["draft","published","archived"]`), value: "published"},
		{name: "dropdown invalid", fieldType: models.FieldTypeDropdown, metadata: []byte(`["draft","published","archived"]`), value: "deleted", wantErr: true},

		// association
		{name: "association valid uuid", fieldType: models.FieldTypeAssociation, value: uuid.New().String()},
		{name: "association invalid", fieldType: models.FieldTypeAssociation, value: "not-a-uuid", wantErr: true},

		// free-form types
		{name: "short_text any value", fieldType: models.FieldTypeShortText, value: "anything goes"},
		{name: "long_text any value", fieldType: models.FieldTypeLongText, value: "anything goes"},
		{name: "file any value", fieldType: models.FieldTypeFile, value: "file-id-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newFieldValueSvc(tt.fieldType, tt.metadata, noopRepo)
			err := set(svc, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !errors.Is(err, service.ErrInvalidInput) {
					t.Errorf("expected ErrInvalidInput, got %v", err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
