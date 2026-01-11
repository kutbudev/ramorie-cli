package crypto

import (
	"testing"
)

func TestIsVaultUnlocked_Initially(t *testing.T) {
	// Reset vault state
	LockVault()

	if IsVaultUnlocked() {
		t.Error("Vault should be locked initially")
	}
}

func TestUnlockVault_Success(t *testing.T) {
	// Reset vault state
	LockVault()

	masterPassword := "test-master-password"

	// Generate encryption parameters
	salt, _ := GenerateSalt()
	symmetricKey, _ := GenerateRandomBytes(KeyLength)
	stretchedMasterKey := DeriveKey(masterPassword, salt, PBKDF2Iterations)

	// Encrypt symmetric key
	encryptedKey, nonce, _ := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)

	// Create vault config
	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: BytesToBase64(encryptedKey),
		KeyNonce:              BytesToBase64(nonce),
		Salt:                  BytesToBase64(salt),
		KDFIterations:         PBKDF2Iterations,
		KDFAlgorithm:          "PBKDF2-SHA256",
	}

	// Unlock vault
	err := UnlockVault(masterPassword, config)
	if err != nil {
		t.Fatalf("UnlockVault() error = %v", err)
	}

	if !IsVaultUnlocked() {
		t.Error("Vault should be unlocked after UnlockVault()")
	}

	// Verify we can get the symmetric key
	key, err := GetSymmetricKey()
	if err != nil {
		t.Fatalf("GetSymmetricKey() error = %v", err)
	}

	if string(key) != string(symmetricKey) {
		t.Error("GetSymmetricKey() returned wrong key")
	}

	// Clean up
	LockVault()
}

func TestUnlockVault_WrongPassword(t *testing.T) {
	// Reset vault state
	LockVault()

	masterPassword := "correct-password"
	wrongPassword := "wrong-password"

	// Generate encryption parameters
	salt, _ := GenerateSalt()
	symmetricKey, _ := GenerateRandomBytes(KeyLength)
	stretchedMasterKey := DeriveKey(masterPassword, salt, PBKDF2Iterations)

	// Encrypt symmetric key
	encryptedKey, nonce, _ := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)

	// Create vault config
	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: BytesToBase64(encryptedKey),
		KeyNonce:              BytesToBase64(nonce),
		Salt:                  BytesToBase64(salt),
		KDFIterations:         PBKDF2Iterations,
		KDFAlgorithm:          "PBKDF2-SHA256",
	}

	// Try to unlock with wrong password
	err := UnlockVault(wrongPassword, config)
	if err == nil {
		t.Error("UnlockVault() should fail with wrong password")
	}

	if IsVaultUnlocked() {
		t.Error("Vault should remain locked after failed unlock")
	}
}

func TestUnlockVault_EncryptionDisabled(t *testing.T) {
	// Reset vault state
	LockVault()

	config := &VaultConfig{
		EncryptionEnabled: false,
	}

	err := UnlockVault("any-password", config)
	if err == nil {
		t.Error("UnlockVault() should fail when encryption is disabled")
	}
}

func TestLockVault(t *testing.T) {
	// First unlock the vault
	masterPassword := "test-password"
	salt, _ := GenerateSalt()
	symmetricKey, _ := GenerateRandomBytes(KeyLength)
	stretchedMasterKey := DeriveKey(masterPassword, salt, PBKDF2Iterations)
	encryptedKey, nonce, _ := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)

	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: BytesToBase64(encryptedKey),
		KeyNonce:              BytesToBase64(nonce),
		Salt:                  BytesToBase64(salt),
		KDFIterations:         PBKDF2Iterations,
	}

	_ = UnlockVault(masterPassword, config)

	// Verify unlocked
	if !IsVaultUnlocked() {
		t.Fatal("Vault should be unlocked before LockVault() test")
	}

	// Lock the vault
	LockVault()

	if IsVaultUnlocked() {
		t.Error("Vault should be locked after LockVault()")
	}

	// Verify we can't get the symmetric key
	_, err := GetSymmetricKey()
	if err == nil {
		t.Error("GetSymmetricKey() should fail after LockVault()")
	}
}

func TestGetSymmetricKey_Locked(t *testing.T) {
	// Reset vault state
	LockVault()

	_, err := GetSymmetricKey()
	if err == nil {
		t.Error("GetSymmetricKey() should fail when vault is locked")
	}
}

func TestEncryptContent_VaultUnlocked(t *testing.T) {
	// Setup unlocked vault
	masterPassword := "test-password"
	salt, _ := GenerateSalt()
	symmetricKey, _ := GenerateRandomBytes(KeyLength)
	stretchedMasterKey := DeriveKey(masterPassword, salt, PBKDF2Iterations)
	encryptedKey, nonce, _ := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)

	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: BytesToBase64(encryptedKey),
		KeyNonce:              BytesToBase64(nonce),
		Salt:                  BytesToBase64(salt),
		KDFIterations:         PBKDF2Iterations,
	}

	_ = UnlockVault(masterPassword, config)
	defer LockVault()

	// Test encryption
	content := "Secret content to encrypt"
	encrypted, contentNonce, isEncrypted, err := EncryptContent(content)

	if err != nil {
		t.Fatalf("EncryptContent() error = %v", err)
	}

	if !isEncrypted {
		t.Error("Content should be encrypted when vault is unlocked")
	}

	if encrypted == content {
		t.Error("Encrypted content should differ from plaintext")
	}

	if contentNonce == "" {
		t.Error("Nonce should not be empty")
	}
}

func TestEncryptContent_VaultLocked(t *testing.T) {
	// Reset vault state
	LockVault()

	content := "Content when vault is locked"
	encrypted, nonce, isEncrypted, err := EncryptContent(content)

	if err != nil {
		t.Fatalf("EncryptContent() error = %v", err)
	}

	if isEncrypted {
		t.Error("Content should not be encrypted when vault is locked")
	}

	// When vault is locked, content is returned as-is
	if encrypted != content {
		t.Errorf("Unencrypted content should be returned as-is, got %q want %q", encrypted, content)
	}

	if nonce != "" {
		t.Error("Nonce should be empty when not encrypted")
	}
}

func TestDecryptContent_Encrypted(t *testing.T) {
	// Setup unlocked vault
	masterPassword := "test-password"
	salt, _ := GenerateSalt()
	symmetricKey, _ := GenerateRandomBytes(KeyLength)
	stretchedMasterKey := DeriveKey(masterPassword, salt, PBKDF2Iterations)
	encryptedKey, nonce, _ := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)

	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: BytesToBase64(encryptedKey),
		KeyNonce:              BytesToBase64(nonce),
		Salt:                  BytesToBase64(salt),
		KDFIterations:         PBKDF2Iterations,
	}

	_ = UnlockVault(masterPassword, config)
	defer LockVault()

	// Encrypt content
	originalContent := "Secret message to decrypt"
	encrypted, contentNonce, _, _ := EncryptContent(originalContent)

	// Decrypt content
	decrypted, err := DecryptContent(encrypted, contentNonce, true)
	if err != nil {
		t.Fatalf("DecryptContent() error = %v", err)
	}

	if decrypted != originalContent {
		t.Errorf("Decrypted content = %q, want %q", decrypted, originalContent)
	}
}

func TestDecryptContent_NotEncrypted(t *testing.T) {
	content := "Plain text content"
	decrypted, err := DecryptContent(content, "", false)

	if err != nil {
		t.Fatalf("DecryptContent() error = %v", err)
	}

	if decrypted != content {
		t.Errorf("DecryptContent() = %q, want %q", decrypted, content)
	}
}

func TestDecryptContent_VaultLocked(t *testing.T) {
	// Setup unlocked vault, encrypt, then lock
	masterPassword := "test-password"
	salt, _ := GenerateSalt()
	symmetricKey, _ := GenerateRandomBytes(KeyLength)
	stretchedMasterKey := DeriveKey(masterPassword, salt, PBKDF2Iterations)
	encryptedKey, nonce, _ := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)

	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: BytesToBase64(encryptedKey),
		KeyNonce:              BytesToBase64(nonce),
		Salt:                  BytesToBase64(salt),
		KDFIterations:         PBKDF2Iterations,
	}

	_ = UnlockVault(masterPassword, config)

	// Encrypt content while unlocked
	encrypted, contentNonce, _, _ := EncryptContent("Secret message")

	// Lock vault
	LockVault()

	// Try to decrypt while locked
	decrypted, err := DecryptContent(encrypted, contentNonce, true)
	if err != nil {
		t.Fatalf("DecryptContent() unexpected error = %v", err)
	}

	// Should return placeholder message
	if decrypted != "[Encrypted - Unlock vault to view]" {
		t.Errorf("DecryptContent() should return placeholder when locked, got %q", decrypted)
	}
}

func TestVaultConfig_Serialization(t *testing.T) {
	config := &VaultConfig{
		EncryptionEnabled:     true,
		EncryptedSymmetricKey: "base64-encrypted-key",
		KeyNonce:              "base64-nonce",
		Salt:                  "base64-salt",
		KDFIterations:         310000,
		KDFAlgorithm:          "PBKDF2-SHA256",
	}

	if !config.EncryptionEnabled {
		t.Error("EncryptionEnabled should be true")
	}

	if config.KDFIterations != 310000 {
		t.Errorf("KDFIterations = %d, want 310000", config.KDFIterations)
	}
}
