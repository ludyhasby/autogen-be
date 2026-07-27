package helperexception

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type Code string

const (
	InvalidArgumentCode  Code = "INVALID_ARGUMENT"  // Represents an invalid argument error.
	NotFoundCode         Code = "NOT_FOUND"         // Represents a not found error.
	AlreadyExistsCode    Code = "ALREADY_EXISTS"    // Represents an already exists error.
	PermissionDeniedCode Code = "PERMISSION_DENIED" // Represents a permission denied error.
	UnauthenticatedCode  Code = "UNAUTHENTICATED"   // Represents an unauthenticated error.
	InternalErrorCode    Code = "INTERNAL"          // Represents an internal error.
)

type Exception struct {
	Code    Code
	Message any
	Err     error
}

func (e *Exception) GetError() *string {
	if e.Err != nil {
		err := e.Err.Error()
		return &err
	}
	return nil
}

func (e *Exception) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("%v", e.Message)
}

func (e *Exception) GetHttpCode() int {
	switch e.Code {
	case InvalidArgumentCode:
		return 400
	case NotFoundCode:
		return 404
	case AlreadyExistsCode:
		return 409
	case PermissionDeniedCode:
		return 403
	case UnauthenticatedCode:
		return 401
	case InternalErrorCode:
		return 500
	default:
		return 500
	}
}

func (e *Exception) IsEqual(err *Exception) bool {
	if err == nil {
		return e == nil
	}
	if e == nil {
		return err == nil
	}
	return e.Code == err.Code && reflect.DeepEqual(e.Message, err.Message)
}

func InvalidArgument(message, dto any) *Exception {
	errorMessage := resolveInvalidArgumentMessage(message, dto)
	return &Exception{
		Code:    InvalidArgumentCode,
		Message: errorMessage,
		Err:     fmt.Errorf("%s", errorMessage),
	}
}

func NotFound(message any) *Exception {
	errorMessage := convertToString(message)
	return &Exception{
		Code:    NotFoundCode,
		Message: errorMessage,
		Err:     fmt.Errorf("%s", errorMessage),
	}
}

func AlreadyExists(message any) *Exception {
	errorMessage := convertToString(message)
	return &Exception{
		Code:    AlreadyExistsCode,
		Message: errorMessage,
		Err:     fmt.Errorf("%s", errorMessage),
	}
}

func PermissionDenied(message any) *Exception {
	errorMessage := convertToString(message)
	return &Exception{
		Code:    PermissionDeniedCode,
		Message: errorMessage,
		Err:     fmt.Errorf("%s", errorMessage),
	}
}

func Unauthenticated(message any) *Exception {
	errorMessage := convertToString(message)
	return &Exception{
		Code:    UnauthenticatedCode,
		Message: errorMessage,
		Err:     fmt.Errorf("%s", errorMessage),
	}
}

func Internal(message any, err error) *Exception {
	errorMessage := convertToString(message)
	return &Exception{
		Code:    InternalErrorCode,
		Message: errorMessage,
		Err:     err,
	}
}

func Conflict(message any) *Exception {
	errorMessage := convertToString(message)
	return &Exception{
		Code:    AlreadyExistsCode,
		Message: errorMessage,
		Err:     fmt.Errorf("%s", errorMessage),
	}
}

func convertToString(message any) string {
	if msg, ok := message.(string); ok {
		return msg
	}
	return fmt.Sprintf("%v", message)
}

func resolveInvalidArgumentMessage(input, dto any) string {
	if input == nil {
		return "invalid argument"
	}

	// validator error
	if ve, ok := input.(validator.ValidationErrors); ok {
		return parseValidatorErrors(ve, dto)
	}

	// general error
	if err, ok := input.(error); ok {
		// check if error wrap
		if ve, ok := errors.Unwrap(err).(validator.ValidationErrors); ok {
			return parseValidatorErrors(ve, dto)
		}
		return err.Error()
	}

	// string
	if s, ok := input.(string); ok {
		return s
	}

	// fallback
	return fmt.Sprintf("%v", input)
}

func parseValidatorErrors(ve validator.ValidationErrors, dto any) string {
	if len(ve) == 0 {
		return "invalid argument"
	}

	fe := ve[0] // get first error

	// get field struct (reflect)
	t := reflect.TypeOf(dto)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if field, ok := t.FieldByName(fe.StructField()); ok {
		if msg := field.Tag.Get("msg"); msg != "" {
			return msg
		}
	}

	// automatic fallback
	return fmt.Sprintf("%s is invalid", fe.Field())
}
