package errors

import (
	"strings"
)

// ParseAPIError extracts user-friendly message from API error
func ParseAPIError(err error) string {
	if err == nil {
		return "Unknown error"
	}

	errStr := err.Error()
	errLower := strings.ToLower(errStr)

	// Rate limiting / Too many requests (429)
	if strings.Contains(errStr, "429") || strings.Contains(errLower, "rate limit") || strings.Contains(errLower, "too many requests") {
		return "⚠️  Rate limit exceeded. Please wait a moment and try again."
	}

	// Account locked (SEC-11)
	if strings.Contains(errLower, "locked") || strings.Contains(errLower, "too many failed") {
		return "🔒 Account is temporarily locked due to too many failed login attempts.\n   Please wait 15 minutes or contact support."
	}

	// Content too large (413)
	if strings.Contains(errStr, "413") || strings.Contains(errLower, "too large") || strings.Contains(errLower, "exceeds") {
		return "📄 Content exceeds maximum allowed length (3M characters / ~750K tokens).\n   Please reduce the content size."
	}

	// Password complexity (SEC-5)
	if strings.Contains(errLower, "password") && (strings.Contains(errLower, "must") || strings.Contains(errLower, "required") || strings.Contains(errLower, "complexity")) {
		return "🔑 Password does not meet security requirements:\n" +
			"   - At least 8 characters\n" +
			"   - At least one uppercase letter\n" +
			"   - At least one lowercase letter\n" +
			"   - At least one number\n" +
			"   - At least one special character (!@#$%^&*)"
	}

	// Authentication errors
	if strings.Contains(errStr, "401") || strings.Contains(errLower, "unauthorized") || strings.Contains(errLower, "invalid api key") {
		return "🔐 Authentication failed. Please run 'ramorie setup' to authenticate."
	}

	// Forbidden / suspended
	if strings.Contains(errStr, "403") || strings.Contains(errLower, "forbidden") || strings.Contains(errLower, "suspended") {
		return "⛔ Access denied. Your account may be suspended. Please contact support."
	}

	// Not found
	if strings.Contains(errStr, "404") || strings.Contains(errLower, "not found") {
		return "🔍 Resource not found. Please check the ID and try again."
	}

	// Server error
	if strings.Contains(errStr, "500") || strings.Contains(errLower, "internal server") {
		return "❌ Server error. Please try again later or contact support if the issue persists."
	}

	// Network errors
	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "connection") || strings.Contains(errLower, "network") {
		return "🌐 Network error. Please check your internet connection and try again."
	}

	// Invalid credentials
	if strings.Contains(errLower, "invalid credentials") {
		return "❌ Invalid email or password. Please try again."
	}

	// User already exists
	if strings.Contains(errLower, "already exists") {
		return "📧 An account with this email already exists. Please login instead."
	}

	// Default: return original error
	return "❌ " + errStr
}

// IsRateLimitError checks if error is rate limit related
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errLower := strings.ToLower(err.Error())
	return strings.Contains(errLower, "429") || strings.Contains(errLower, "rate limit")
}

// IsAuthError checks if error is authentication related
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	errLower := strings.ToLower(err.Error())
	return strings.Contains(errLower, "401") || strings.Contains(errLower, "unauthorized") || strings.Contains(errLower, "api key")
}

// IsContentTooLargeError checks if error is content size related
func IsContentTooLargeError(err error) bool {
	if err == nil {
		return false
	}
	errLower := strings.ToLower(err.Error())
	return strings.Contains(errLower, "413") || strings.Contains(errLower, "too large") || strings.Contains(errLower, "exceeds")
}

