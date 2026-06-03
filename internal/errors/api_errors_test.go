package errors

import (
	"errors"
	"strings"
	"testing"
)

// The backend ENCRYPTION_REQUIRED message contains the word "unlock", which
// historically caused ParseAPIError to misclassify it as an account-lockout
// (the generic "locked" branch). These tests guard the dedicated branch.
func TestIsEncryptionRequiredError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"code", errors.New("request failed: status 400 body {\"code\":\"ENCRYPTION_REQUIRED\"}"), true},
		{"human message", errors.New("This project requires encryption. CLI/MCP: run 'ramorie setup unlock'."), true},
		{"unrelated lock", errors.New("account is locked due to too many failed login attempts"), false},
		{"unrelated 404", errors.New("status 404 not found"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsEncryptionRequiredError(tc.err); got != tc.want {
				t.Fatalf("IsEncryptionRequiredError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestParseAPIError_EncryptionRequiredNotMisclassifiedAsLocked(t *testing.T) {
	// Verbatim backend message — contains "unlock".
	err := errors.New("This project requires encryption. CLI/MCP: run 'ramorie setup unlock'. (status 400, code ENCRYPTION_REQUIRED)")
	msg := ParseAPIError(err)

	if strings.Contains(msg, "failed login attempts") {
		t.Fatalf("ENCRYPTION_REQUIRED misclassified as account-lockout: %q", msg)
	}
	if !strings.Contains(msg, "set-encryption") {
		t.Fatalf("ParseAPIError should point at set-encryption; got %q", msg)
	}
}

func TestEncryptionRequiredMessage_StateAware(t *testing.T) {
	// THE bug case: account encryption disabled, but project demands it.
	// Unlocking can't help — message must say so and point at set-encryption.
	disabled := EncryptionRequiredMessage("myproj", true /*vaultUnlocked*/, false /*accountEncryptionEnabled*/)
	if !strings.Contains(disabled, "conflict") {
		t.Errorf("disabled-account case must flag the conflict; got %q", disabled)
	}
	if !strings.Contains(disabled, "set-encryption myproj false") {
		t.Errorf("disabled-account case must give the exact fix command; got %q", disabled)
	}
	if !strings.Contains(disabled, "NOT help") {
		t.Errorf("disabled-account case must say unlocking won't help; got %q", disabled)
	}

	// Account encryption ON, vault locked → unlock genuinely helps.
	locked := EncryptionRequiredMessage("myproj", false /*vaultUnlocked*/, true /*accountEncryptionEnabled*/)
	if !strings.Contains(locked, "setup unlock") {
		t.Errorf("locked-vault case should suggest unlock; got %q", locked)
	}
	if strings.Contains(locked, "conflict") {
		t.Errorf("locked-vault case should NOT mention a conflict; got %q", locked)
	}

	// Project name with spaces must be quoted in the suggested command.
	spaced := EncryptionRequiredMessage("my project", true, false)
	if !strings.Contains(spaced, "\"my project\"") {
		t.Errorf("project name with spaces must be quoted; got %q", spaced)
	}

	// Empty project name → placeholder.
	empty := EncryptionRequiredMessage("", true, false)
	if !strings.Contains(empty, "<project>") {
		t.Errorf("empty project name must use placeholder; got %q", empty)
	}
}
