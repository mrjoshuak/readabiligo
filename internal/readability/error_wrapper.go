package readability

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorType defines the category of an error
type ErrorType string

// Error types
const (
	ParseError       ErrorType = "parse"
	ExtractionError  ErrorType = "extraction"
	ValidationError  ErrorType = "validation"
	CleanupError     ErrorType = "cleanup"
	TimeoutError     ErrorType = "timeout"
	MetadataError    ErrorType = "metadata"
)

// Common errors that can be used throughout the package
var (
	ErrNoDocument    = errors.New("no document to parse")
	ErrDocumentLarge = errors.New("document too large")
	ErrNoContent     = errors.New("could not extract article content")
	ErrTimeout       = errors.New("operation timed out")
)

// WrapError wraps an error with context information
func WrapError(err error, errorType ErrorType, funcName, message string) error {
	if err == nil {
		return nil
	}
	
	// If the error doesn't have a message, use the provided one
	if message == "" {
		return fmt.Errorf("[%s:%s] %w", errorType, funcName, err)
	}
	
	return fmt.Errorf("[%s:%s] %s: %w", errorType, funcName, message, err)
}

// WrapParseError wraps a parsing error
func WrapParseError(err error, funcName, message string) error {
	return WrapError(err, ParseError, funcName, message)
}

// WrapExtractionError wraps an extraction error
func WrapExtractionError(err error, funcName, message string) error {
	return WrapError(err, ExtractionError, funcName, message)
}

// WrapValidationError wraps a validation error
func WrapValidationError(err error, funcName, message string) error {
	return WrapError(err, ValidationError, funcName, message)
}

// WrapCleanupError wraps a cleanup error
func WrapCleanupError(err error, funcName, message string) error {
	return WrapError(err, CleanupError, funcName, message)
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	if err == nil {
		return false
	}
	
	// Simple string-based check for the error type in the error message
	return strings.Contains(err.Error(), fmt.Sprintf("[%s:", errorType))
}

// IsParseError returns true if the error is a parse error
func IsParseError(err error) bool {
	return IsErrorType(err, ParseError)
}

// IsExtractionError returns true if the error is an extraction error
func IsExtractionError(err error) bool {
	return IsErrorType(err, ExtractionError)
}

// IsValidationError returns true if the error is a validation error
func IsValidationError(err error) bool {
	return IsErrorType(err, ValidationError)
}

// IsCleanupError returns true if the error is a cleanup error
func IsCleanupError(err error) bool {
	return IsErrorType(err, CleanupError)
}