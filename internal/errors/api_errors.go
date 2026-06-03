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

	// Per-project encryption requirement (ENCRYPTION_REQUIRED). Must be
	// checked BEFORE the generic "locked" branch below: the backend message
	// contains the word "unlock", which would otherwise be misclassified as
	// an account-lockout. Write paths should prefer EncryptionRequiredMessage
	// (state-aware); this is the safe fallback when state isn't available.
	if IsEncryptionRequiredError(err) {
		return "🔒 This project requires encrypted writes, but the content was sent unencrypted.\n" +
			"   If your account encryption is enabled: run `ramorie setup unlock`.\n" +
			"   If your account encryption is DISABLED (the two conflict): run\n" +
			"   `ramorie project set-encryption <project> false` to allow plaintext writes."
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

// IsEncryptionRequiredError reports whether the backend rejected a write
// because the target project has encryption_required=true. Matches the
// backend's machine-readable code (ENCRYPTION_REQUIRED) and its human
// message as a fallback.
func IsEncryptionRequiredError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if strings.Contains(s, "ENCRYPTION_REQUIRED") {
		return true
	}
	l := strings.ToLower(s)
	return strings.Contains(l, "encryption required") || strings.Contains(l, "this project requires encryption")
}

// EncryptionRequiredMessage builds a state-aware message for an
// ENCRYPTION_REQUIRED write rejection. The generic ParseAPIError text tells
// the user to run `ramorie setup unlock`, which is actively wrong when the
// account's encryption is DISABLED — unlocking can't satisfy a project that
// demands encrypted writes while the account refuses to encrypt. This
// surfaces the real cause and the actual fix.
//
//   - projectName: resolved name for a friendlier message ("" → "this project")
//   - vaultUnlocked: local crypto vault state
//   - accountEncryptionEnabled: the EFFECTIVE server-side encryption flag
//
// The contradiction case (vault unlocked OR account encryption disabled, yet
// the project still demands encryption) is the one the user hit.
func EncryptionRequiredMessage(projectName string, vaultUnlocked, accountEncryptionEnabled bool) string {
	name := projectName
	if name == "" {
		name = "this project"
	} else {
		name = "'" + name + "'"
	}

	// Account encryption is OFF but the project still requires encryption →
	// unsatisfiable. This is THE misleading-error case.
	if !accountEncryptionEnabled {
		return "🔒 Project " + name + " requires encrypted writes, but your account encryption is DISABLED — these conflict.\n" +
			"   Unlocking the vault will NOT help. To resolve, either:\n" +
			"     • allow plaintext writes:  ramorie project set-encryption " + projectArg(projectName) + " false\n" +
			"     • or re-enable account encryption in the web app, then `ramorie setup unlock`."
	}

	// Account encryption is ON but the vault is locked → unlock genuinely helps.
	if !vaultUnlocked {
		return "🔒 Project " + name + " requires encrypted writes and your vault is locked.\n" +
			"   Run `ramorie setup unlock` to unlock it, then retry."
	}

	// Account encryption ON, vault unlocked, but write still rejected as
	// plaintext — likely an org/scope mismatch in the encryption decision.
	return "🔒 Project " + name + " requires encrypted writes but the content was sent unencrypted.\n" +
		"   If you intend this project to allow plaintext, run\n" +
		"   `ramorie project set-encryption " + projectArg(projectName) + " false`."
}

// projectArg returns a shell-friendly placeholder for the project argument.
func projectArg(projectName string) string {
	if projectName == "" {
		return "<project>"
	}
	if strings.ContainsAny(projectName, " \t") {
		return "\"" + projectName + "\""
	}
	return projectName
}
