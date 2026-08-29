// Package validate wraps go-playground/validator with a single entry point
// that turns the first failing `validate:` struct tag into a human-readable
// message. Handlers call validate.Struct(&req) right after BodyParser instead
// of hand-writing field checks, so the rules live on the DTO next to the JSON
// shape and stay consistent across every module.
package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var v = newValidator()

func newValidator() *validator.Validate {
	val := validator.New(validator.WithRequiredStructEnabled())
	// Report the JSON field name, not the Go struct field name — that is what
	// the client actually sent and what the OpenAPI spec documents.
	val.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})
	return val
}

// Struct validates s against its `validate:` tags. It returns "" when the
// value is valid, otherwise a message naming the first offending field —
// suitable to hand straight to response.BadRequest.
func Struct(s any) string {
	err := v.Struct(s)
	if err == nil {
		return ""
	}

	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) || len(verrs) == 0 {
		return "invalid request body"
	}
	return message(verrs[0])
}

func message(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", e.Field(), strings.ReplaceAll(e.Param(), " ", ", "))
	case "min":
		return fmt.Sprintf("%s must be at least %s", e.Field(), e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", e.Field(), e.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", e.Field(), e.Param())
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", e.Field(), e.Param())
	case "uuid", "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", e.Field())
	default:
		return fmt.Sprintf("%s is invalid", e.Field())
	}
}
