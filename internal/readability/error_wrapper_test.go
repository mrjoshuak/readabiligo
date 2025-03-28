package readability

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapError(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := WrapError(baseErr, ParseError, "TestFunc", "test message")
	
	// Test that the error is properly formatted
	if !strings.Contains(wrapped.Error(), "[parse:TestFunc]") {
		t.Errorf("Error message should contain formatted prefix, got: %s", wrapped.Error())
	}
	
	// Test that the message is included
	if !strings.Contains(wrapped.Error(), "test message") {
		t.Errorf("Error message should contain the message, got: %s", wrapped.Error())
	}
	
	// Test that the original error is included
	if !strings.Contains(wrapped.Error(), baseErr.Error()) {
		t.Errorf("Error message should contain the original error, got: %s", wrapped.Error())
	}
	
	// Test unwrapping
	if !errors.Is(wrapped, baseErr) {
		t.Errorf("errors.Is should return true for the base error")
	}
}

func TestWrapErrorSpecificTypes(t *testing.T) {
	baseErr := errors.New("base error")
	
	tests := []struct {
		name       string
		wrapFunc   func(error, string, string) error
		errorType  ErrorType
		checkFunc  func(error) bool
	}{
		{
			name:       "ParseError",
			wrapFunc:   WrapParseError,
			errorType:  ParseError,
			checkFunc:  IsParseError,
		},
		{
			name:       "ExtractionError",
			wrapFunc:   WrapExtractionError,
			errorType:  ExtractionError,
			checkFunc:  IsExtractionError,
		},
		{
			name:       "ValidationError",
			wrapFunc:   WrapValidationError,
			errorType:  ValidationError,
			checkFunc:  IsValidationError,
		},
		{
			name:       "CleanupError",
			wrapFunc:   WrapCleanupError,
			errorType:  CleanupError,
			checkFunc:  IsCleanupError,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedErr := tt.wrapFunc(baseErr, "TestFunc", "test message")
			
			// Test specific error type detection
			if !tt.checkFunc(wrappedErr) {
				t.Errorf("%s should be detected by Is%s", tt.name, tt.name)
			}
			
			// Test error type checking function
			if !IsErrorType(wrappedErr, tt.errorType) {
				t.Errorf("IsErrorType should identify %s as type %s", tt.name, tt.errorType)
			}
			
			// Test error message formatting
			if !strings.Contains(wrappedErr.Error(), string(tt.errorType)) {
				t.Errorf("Error message should contain error type %s", tt.errorType)
			}
		})
	}
}