package crypto

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "ramorie-vault"
	keyringUser    = "symmetric-key"
)

var (
	// fallbackMode indicates if we're using file-based fallback (headless systems)
	fallbackMode     bool
	fallbackModeMu   sync.RWMutex
	fallbackChecked  bool
)

// checkKeyringAvailable tests if system keyring is available
func checkKeyringAvailable() bool {
	fallbackModeMu.Lock()
	defer fallbackModeMu.Unlock()

	if fallbackChecked {
		return !fallbackMode
	}

	// Try to access keyring with a test operation
	testKey := "ramorie-keyring-test"
	err := keyring.Set(keyringService, testKey, "test")
	if err != nil {
		fallbackMode = true
		fallbackChecked = true
		return false
	}

	// Clean up test key
	_ = keyring.Delete(keyringService, testKey)
	fallbackChecked = true
	return true
}

// isFallbackMode returns true if using file-based fallback
func isFallbackMode() bool {
	fallbackModeMu.RLock()
	defer fallbackModeMu.RUnlock()
	return fallbackMode
}

// getFallbackPath returns the path for fallback key storage
func getFallbackPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ramorie", ".vault.session"), nil
}

// StoreSymmetricKey stores the symmetric key in system keyring or fallback file
func StoreSymmetricKey(key []byte) error {
	encoded := base64.StdEncoding.EncodeToString(key)

	// Check if keyring is available
	if checkKeyringAvailable() {
		err := keyring.Set(keyringService, keyringUser, encoded)
		if err != nil {
			return fmt.Errorf("failed to store key in keyring: %w", err)
		}
		return nil
	}

	// Fallback to file-based storage
	return storeFallbackKey(encoded)
}

// RetrieveSymmetricKey retrieves the symmetric key from system keyring or fallback file
func RetrieveSymmetricKey() ([]byte, error) {
	var encoded string
	var err error

	if !isFallbackMode() && checkKeyringAvailable() {
		encoded, err = keyring.Get(keyringService, keyringUser)
		if err != nil {
			// Key not found in keyring
			return nil, fmt.Errorf("key not found in keyring: %w", err)
		}
	} else {
		// Try fallback file
		encoded, err = retrieveFallbackKey()
		if err != nil {
			return nil, fmt.Errorf("key not found in fallback: %w", err)
		}
	}

	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	return key, nil
}

// DeleteSymmetricKey removes the symmetric key from system keyring or fallback file
func DeleteSymmetricKey() error {
	var keyringErr, fallbackErr error

	// Try to delete from keyring (if available)
	if !isFallbackMode() {
		keyringErr = keyring.Delete(keyringService, keyringUser)
	}

	// Also try to delete fallback file (in case it exists)
	fallbackErr = deleteFallbackKey()

	// If both failed, return an error
	if keyringErr != nil && fallbackErr != nil {
		return fmt.Errorf("failed to delete key from keyring and fallback")
	}

	return nil
}

// HasStoredKey checks if there's a stored symmetric key available
func HasStoredKey() bool {
	// Check keyring first
	if !isFallbackMode() && checkKeyringAvailable() {
		_, err := keyring.Get(keyringService, keyringUser)
		if err == nil {
			return true
		}
	}

	// Check fallback file
	path, err := getFallbackPath()
	if err != nil {
		return false
	}

	_, err = os.Stat(path)
	return err == nil
}

// Fallback file operations for headless systems

func storeFallbackKey(encoded string) error {
	path, err := getFallbackPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write key with restrictive permissions (owner read/write only)
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to write fallback key: %w", err)
	}

	return nil
}

func retrieveFallbackKey() (string, error) {
	path, err := getFallbackPath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func deleteFallbackKey() error {
	path, err := getFallbackPath()
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// GetStorageMode returns a string describing current storage mode
func GetStorageMode() string {
	if !fallbackChecked {
		checkKeyringAvailable()
	}

	if isFallbackMode() {
		return "file-based (keyring unavailable)"
	}
	return "system-keyring"
}
