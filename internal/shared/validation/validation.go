package validation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/anjal-niraula/engage-engine/internal/shared/errors"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register tag name function to extract the json tag as field name
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Validator returns the configured validator instance.
func Validator() *validator.Validate {
	return validate
}

// DecodeAndValidate decodes the request body as JSON into target and performs struct validation.
func DecodeAndValidate(r *http.Request, target any) *errors.HTTPError {
	if r.Body == nil {
		return errors.NewBadRequestError("Request body cannot be empty", false, nil, nil, nil)
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			return errors.NewBadRequestError("Request body cannot be empty", false, nil, nil, nil)
		}
		return errors.NewBadRequestError(fmt.Sprintf("Malformed JSON body: %v", err), false, nil, nil, nil)
	}

	return ValidateStruct(target)
}

// ValidateStruct validates a struct instance using go-playground/validator/v10 tags.
func ValidateStruct(target any) *errors.HTTPError {
	if err := validate.Struct(target); err != nil {
		if valErrs, ok := err.(validator.ValidationErrors); ok {
			fieldErrors := make([]errors.FieldError, 0, len(valErrs))
			for _, fe := range valErrs {
				field := fe.Field()
				if field == "" {
					field = strings.ToLower(fe.StructField())
				}

				fieldErrors = append(fieldErrors, errors.FieldError{
					Field: field,
					Error: formatValidationMessage(fe),
				})
			}
			return errors.NewBadRequestError("Validation failed", false, nil, fieldErrors, nil)
		}

		return errors.NewBadRequestError(fmt.Sprintf("Validation error: %v", err), false, nil, nil, nil)
	}

	return nil
}

func formatValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must not exceed %s characters", fe.Param())
		}
		return fmt.Sprintf("must not exceed %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "uuid":
		return "must be a valid UUID"
	case "url":
		return "must be a valid URL"
	default:
		if fe.Param() != "" {
			return fmt.Sprintf("failed validation on rule '%s:%s'", fe.Tag(), fe.Param())
		}
		return fmt.Sprintf("failed validation on rule '%s'", fe.Tag())
	}
}

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsValidUUID tests whether a given string is a valid RFC 4122 UUID.
func IsValidUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}

