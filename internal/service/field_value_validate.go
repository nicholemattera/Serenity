package service

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
)

var (
	phonePattern = regexp.MustCompile(`^\+?[\d\s\-().]{7,20}$`)
	colorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
)

// validateFieldValue checks that value is acceptable for the given field type.
// Returns a wrapped ErrInvalidInput on failure.
func validateFieldValue(field *models.Field, value string) error {
	wrap := func(msg string) error {
		return fmt.Errorf("%w: %s", ErrInvalidInput, msg)
	}

	switch field.Type {
	case models.FieldTypeCheckbox:
		if value != "true" && value != "false" {
			return wrap("checkbox value must be \"true\" or \"false\"")
		}

	case models.FieldTypeNumber:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return wrap("number value must be a valid number")
		}

	case models.FieldTypeEmail:
		if _, err := mail.ParseAddress(value); err != nil {
			return wrap("email value must be a valid email address")
		}

	case models.FieldTypeURL:
		u, err := url.ParseRequestURI(value)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return wrap("url value must be a valid URL with scheme and host")
		}

	case models.FieldTypePhone:
		if !phonePattern.MatchString(value) {
			return wrap("phone value must be a valid phone number")
		}

	case models.FieldTypeColor:
		if !colorPattern.MatchString(value) {
			return wrap("color value must be a hex color code (#RGB or #RRGGBB)")
		}

	case models.FieldTypeDate:
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return wrap("date value must be in YYYY-MM-DD format")
		}

	case models.FieldTypeDateTime:
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return wrap("datetime value must be in RFC3339 format (e.g. 2006-01-02T15:04:05Z)")
		}

	case models.FieldTypeTime:
		parsed := false
		for _, layout := range []string{"15:04", "15:04:05"} {
			if _, err := time.Parse(layout, value); err == nil {
				parsed = true
				break
			}
		}
		if !parsed {
			return wrap("time value must be in HH:MM or HH:MM:SS format")
		}

	case models.FieldTypeDropdown:
		var options []string
		if err := json.Unmarshal(field.Metadata, &options); err != nil {
			return wrap("dropdown field has invalid metadata")
		}
		for _, opt := range options {
			if opt == value {
				return nil
			}
		}
		return wrap(fmt.Sprintf("dropdown value must be one of: %s", strings.Join(options, ", ")))

	case models.FieldTypeAssociation:
		if _, err := uuid.Parse(value); err != nil {
			return wrap("association value must be a valid UUID")
		}

	case models.FieldTypeShortText, models.FieldTypeFile:
		// No format constraint; any non-empty string is valid.

	case models.FieldTypeLongText:
		// No format constraint.
	}

	return nil
}
