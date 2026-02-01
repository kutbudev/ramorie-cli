package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// Constants matching frontend implementation
const (
	SaltLength       = 16  // 128-bit salt
	KeyLength        = 32  // AES-256
	NonceLength      = 12  // GCM nonce
	PBKDF2Iterations = 310000 // OWASP 2025 recommendation
)

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// GenerateSalt generates a 16-byte (128-bit) salt
func GenerateSalt() ([]byte, error) {
	return GenerateRandomBytes(SaltLength)
}

// GenerateNonce generates a 12-byte nonce for AES-GCM
func GenerateNonce() ([]byte, error) {
	return GenerateRandomBytes(NonceLength)
}

// DeriveKey derives a 256-bit key from password using PBKDF2-SHA256
func DeriveKey(password string, salt []byte, iterations int) []byte {
	return pbkdf2.Key([]byte(password), salt, iterations, KeyLength, sha256.New)
}

// DeriveKeyWithDefaults derives a key using default PBKDF2 iterations
func DeriveKeyWithDefaults(password string, salt []byte) []byte {
	return DeriveKey(password, salt, PBKDF2Iterations)
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns ciphertext and nonce
func Encrypt(plaintext string, key []byte) (ciphertext, nonce []byte, err error) {
	if len(key) != KeyLength {
		return nil, nil, fmt.Errorf("invalid key length: expected %d, got %d", KeyLength, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err = GenerateNonce()
	if err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func Decrypt(ciphertext, nonce, key []byte) (string, error) {
	if len(key) != KeyLength {
		return "", fmt.Errorf("invalid key length: expected %d, got %d", KeyLength, len(key))
	}

	if len(nonce) != NonceLength {
		return "", fmt.Errorf("invalid nonce length: expected %d, got %d", NonceLength, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// EncryptToBase64 encrypts plaintext and returns base64-encoded ciphertext and nonce
func EncryptToBase64(plaintext string, key []byte) (ciphertextB64, nonceB64 string, err error) {
	ciphertext, nonce, err := Encrypt(plaintext, key)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce), nil
}

// DecryptFromBase64 decrypts base64-encoded ciphertext
func DecryptFromBase64(ciphertextB64, nonceB64 string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	return Decrypt(ciphertext, nonce, key)
}

// Hash computes SHA-256 hash of data
func Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// BytesToHex converts bytes to hex string
func BytesToHex(data []byte) string {
	return fmt.Sprintf("%x", data)
}

// BytesToBase64 converts bytes to base64 string
func BytesToBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64ToBytes converts base64 string to bytes
func Base64ToBytes(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// CreatePasswordHash creates a hash of the stretched master key for server verification
// This matches the frontend's createMasterPasswordHash function
func CreatePasswordHash(password string, salt []byte, iterations int) string {
	// First derive the stretched master key
	stretchedKey := DeriveKey(password, salt, iterations)
	// Then hash it once more with the password for the server
	finalHash := pbkdf2.Key(stretchedKey, []byte(password), 1, KeyLength, sha256.New)
	return BytesToHex(finalHash)
}

// EncryptSymmetricKey encrypts a symmetric key with the stretched master key
func EncryptSymmetricKey(symmetricKey, stretchedMasterKey []byte) (encryptedKey, nonce []byte, err error) {
	return Encrypt(string(symmetricKey), stretchedMasterKey)
}

// DecryptSymmetricKey decrypts the protected symmetric key with the stretched master key
func DecryptSymmetricKey(encryptedKey, nonce, stretchedMasterKey []byte) ([]byte, error) {
	decrypted, err := Decrypt(encryptedKey, nonce, stretchedMasterKey)
	if err != nil {
		return nil, err
	}
	return []byte(decrypted), nil
}

// ============================================================================
// CONTENT HASH - For duplicate detection before encryption
// ============================================================================

// ComputeContentHash computes a SHA-256 hash of normalized content for duplicate detection.
// This should be called BEFORE encryption so the hash represents the plaintext.
// The normalization ensures that minor formatting differences don't create different hashes.
func ComputeContentHash(content string) string {
	normalized := normalizeForHash(content)
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}

// normalizeForHash normalizes content for consistent hash computation
// - Converts to lowercase
// - Collapses multiple whitespace to single space
// - Trims leading/trailing whitespace
func normalizeForHash(content string) string {
	// Convert to lowercase
	normalized := strings.ToLower(content)

	// Replace multiple whitespace (spaces, tabs, newlines) with single space
	whitespaceRegex := regexp.MustCompile(`\s+`)
	normalized = whitespaceRegex.ReplaceAllString(normalized, " ")

	// Trim leading/trailing whitespace
	normalized = strings.TrimSpace(normalized)

	return normalized
}
