package crypto

import (
	"testing"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if len(salt1) != SaltLength {
		t.Errorf("GenerateSalt() length = %d, want %d", len(salt1), SaltLength)
	}

	// Generate another salt - should be different (uniqueness)
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if string(salt1) == string(salt2) {
		t.Error("GenerateSalt() generated duplicate salts")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce1, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce() error = %v", err)
	}

	if len(nonce1) != NonceLength {
		t.Errorf("GenerateNonce() length = %d, want %d", len(nonce1), NonceLength)
	}

	// Generate another nonce - should be different (uniqueness)
	nonce2, err := GenerateNonce()
	if err != nil {
		t.Fatalf("GenerateNonce() error = %v", err)
	}

	if string(nonce1) == string(nonce2) {
		t.Error("GenerateNonce() generated duplicate nonces")
	}
}

func TestDeriveKey(t *testing.T) {
	password := "test-password-123"
	salt, _ := GenerateSalt()

	// Test consistency - same inputs should produce same output
	key1 := DeriveKey(password, salt, PBKDF2Iterations)
	key2 := DeriveKey(password, salt, PBKDF2Iterations)

	if string(key1) != string(key2) {
		t.Error("DeriveKey() not consistent for same inputs")
	}

	if len(key1) != KeyLength {
		t.Errorf("DeriveKey() length = %d, want %d", len(key1), KeyLength)
	}

	// Test different passwords produce different keys
	differentKey := DeriveKey("different-password", salt, PBKDF2Iterations)
	if string(key1) == string(differentKey) {
		t.Error("DeriveKey() produced same key for different passwords")
	}

	// Test different salts produce different keys
	differentSalt, _ := GenerateSalt()
	differentSaltKey := DeriveKey(password, differentSalt, PBKDF2Iterations)
	if string(key1) == string(differentSaltKey) {
		t.Error("DeriveKey() produced same key for different salts")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	testCases := []struct {
		name      string
		plaintext string
	}{
		{"simple", "Hello, World!"},
		{"empty", ""},
		{"unicode", "こんにちは世界 🔐 مرحبا"},
		{"long", "This is a longer piece of text that we want to encrypt and decrypt to ensure the algorithm works correctly with various input sizes."},
		{"special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	key, _ := GenerateRandomBytes(KeyLength)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			ciphertext, nonce, err := Encrypt(tc.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			// Decrypt
			decrypted, err := Decrypt(ciphertext, nonce, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Roundtrip failed: got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestEncryptDecrypt_WrongKey(t *testing.T) {
	plaintext := "Secret message"
	correctKey, _ := GenerateRandomBytes(KeyLength)
	wrongKey, _ := GenerateRandomBytes(KeyLength)

	ciphertext, nonce, err := Encrypt(plaintext, correctKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// Attempt to decrypt with wrong key should fail
	_, err = Decrypt(ciphertext, nonce, wrongKey)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

func TestEncryptDecrypt_InvalidKeyLength(t *testing.T) {
	plaintext := "Test"
	shortKey := []byte("short")

	_, _, err := Encrypt(plaintext, shortKey)
	if err == nil {
		t.Error("Encrypt() should fail with invalid key length")
	}
}

func TestEncryptToBase64_DecryptFromBase64(t *testing.T) {
	plaintext := "Test message for base64 encoding"
	key, _ := GenerateRandomBytes(KeyLength)

	// Encrypt to base64
	ciphertextB64, nonceB64, err := EncryptToBase64(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptToBase64() error = %v", err)
	}

	if ciphertextB64 == "" || nonceB64 == "" {
		t.Error("EncryptToBase64() returned empty values")
	}

	// Decrypt from base64
	decrypted, err := DecryptFromBase64(ciphertextB64, nonceB64, key)
	if err != nil {
		t.Fatalf("DecryptFromBase64() error = %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Base64 roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestBytesToBase64_Base64ToBytes(t *testing.T) {
	original := []byte("Test data for encoding")

	// Encode
	encoded := BytesToBase64(original)
	if encoded == "" {
		t.Error("BytesToBase64() returned empty string")
	}

	// Decode
	decoded, err := Base64ToBytes(encoded)
	if err != nil {
		t.Fatalf("Base64ToBytes() error = %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("Base64 roundtrip failed: got %v, want %v", decoded, original)
	}
}

func TestCreatePasswordHash(t *testing.T) {
	password := "master-password"
	salt, _ := GenerateSalt()

	hash1 := CreatePasswordHash(password, salt, PBKDF2Iterations)
	hash2 := CreatePasswordHash(password, salt, PBKDF2Iterations)

	// Same inputs should produce same hash
	if hash1 != hash2 {
		t.Error("CreatePasswordHash() not consistent for same inputs")
	}

	// Hash should be hex-encoded (64 chars for 32 bytes)
	if len(hash1) != 64 {
		t.Errorf("CreatePasswordHash() length = %d, want 64", len(hash1))
	}

	// Different password should produce different hash
	differentHash := CreatePasswordHash("different-password", salt, PBKDF2Iterations)
	if hash1 == differentHash {
		t.Error("CreatePasswordHash() produced same hash for different passwords")
	}
}

func TestEncryptSymmetricKey_DecryptSymmetricKey(t *testing.T) {
	// Generate symmetric key to protect
	symmetricKey, _ := GenerateRandomBytes(KeyLength)

	// Generate stretched master key
	salt, _ := GenerateSalt()
	stretchedMasterKey := DeriveKey("master-password", salt, PBKDF2Iterations)

	// Encrypt the symmetric key
	encryptedKey, nonce, err := EncryptSymmetricKey(symmetricKey, stretchedMasterKey)
	if err != nil {
		t.Fatalf("EncryptSymmetricKey() error = %v", err)
	}

	// Decrypt the symmetric key
	decryptedKey, err := DecryptSymmetricKey(encryptedKey, nonce, stretchedMasterKey)
	if err != nil {
		t.Fatalf("DecryptSymmetricKey() error = %v", err)
	}

	if string(decryptedKey) != string(symmetricKey) {
		t.Error("Symmetric key roundtrip failed")
	}
}

func TestHash(t *testing.T) {
	data := []byte("test data")

	hash1 := Hash(data)
	hash2 := Hash(data)

	// Same input should produce same hash
	if string(hash1) != string(hash2) {
		t.Error("Hash() not consistent for same input")
	}

	// Hash should be 32 bytes (SHA-256)
	if len(hash1) != 32 {
		t.Errorf("Hash() length = %d, want 32", len(hash1))
	}

	// Different input should produce different hash
	differentHash := Hash([]byte("different data"))
	if string(hash1) == string(differentHash) {
		t.Error("Hash() produced same hash for different inputs")
	}
}
