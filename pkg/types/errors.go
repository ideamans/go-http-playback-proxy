package types

import (
	"errors"
	"fmt"
)

// ErrorType represents different types of errors in the proxy
type ErrorType string

const (
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeInventory  ErrorType = "inventory"
	ErrorTypeEncoding   ErrorType = "encoding"
	ErrorTypeFormat     ErrorType = "format"
	ErrorTypeFilesystem ErrorType = "filesystem"
	ErrorTypeValidation ErrorType = "validation"
)

// ProxyError represents a structured error with type and context
type ProxyError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *ProxyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// Unwrap allows errors.Is and errors.As to work
func (e *ProxyError) Unwrap() error {
	return e.Cause
}

// Is allows comparison with other errors
func (e *ProxyError) Is(target error) bool {
	if target == nil {
		return false
	}
	
	var targetErr *ProxyError
	if errors.As(target, &targetErr) {
		return e.Type == targetErr.Type
	}
	
	return errors.Is(e.Cause, target)
}

// NewProxyError creates a new ProxyError
func NewProxyError(errType ErrorType, message string, cause error) *ProxyError {
	return &ProxyError{
		Type:    errType,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context information to the error
func (e *ProxyError) WithContext(key string, value interface{}) *ProxyError {
	e.Context[key] = value
	return e
}

// Common error constructors
func NewNetworkError(message string, cause error) *ProxyError {
	return NewProxyError(ErrorTypeNetwork, message, cause)
}

func NewInventoryError(message string, cause error) *ProxyError {
	return NewProxyError(ErrorTypeInventory, message, cause)
}

func NewEncodingError(message string, cause error) *ProxyError {
	return NewProxyError(ErrorTypeEncoding, message, cause)
}

func NewFormatError(message string, cause error) *ProxyError {
	return NewProxyError(ErrorTypeFormat, message, cause)
}

func NewFilesystemError(message string, cause error) *ProxyError {
	return NewProxyError(ErrorTypeFilesystem, message, cause)
}

func NewValidationError(message string, cause error) *ProxyError {
	return NewProxyError(ErrorTypeValidation, message, cause)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errType ErrorType) bool {
	var proxyErr *ProxyError
	if errors.As(err, &proxyErr) {
		return proxyErr.Type == errType
	}
	return false
}