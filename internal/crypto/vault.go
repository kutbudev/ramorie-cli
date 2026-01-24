package crypto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// VaultState represents the current state of the encryption vault
type VaultState struct {
	IsUnlocked   bool
	SymmetricKey []byte
	Salt         []byte
	Iterations   int
	// Organization encryption keys (orgId -> derived org key bytes)
	OrgKeys map[string][]byte
}

// OrgVaultConfig stores organization encryption configuration
type OrgVaultConfig struct {
	OrgID         string `json:"org_id"`
	Salt          string `json:"salt"`           // base64
	KDFIterations int    `json:"kdf_iterations"`
	KDFAlgorithm  string `json:"kdf_algorithm"`
	IsEnabled     bool   `json:"is_enabled"`
}

// VaultConfig stores encryption configuration (persisted to disk)
type VaultConfig struct {
	EncryptionEnabled     bool   `json:"encryption_enabled"`
	EncryptedSymmetricKey string `json:"encrypted_symmetric_key"` // base64
	KeyNonce              string `json:"key_nonce"`               // base64
	Salt                  string `json:"salt"`                    // base64
	KDFIterations         int    `json:"kdf_iterations"`
	KDFAlgorithm          string `json:"kdf_algorithm"`
}

var (
	currentVault *VaultState
	vaultMutex   sync.RWMutex
)

// IsVaultUnlocked returns whether the vault is currently unlocked
// Checks both in-memory state and system keyring for cross-process support
func IsVaultUnlocked() bool {
	vaultMutex.RLock()
	if currentVault != nil && currentVault.IsUnlocked {
		vaultMutex.RUnlock()
		return true
	}
	vaultMutex.RUnlock()

	// Check if key exists in system keyring (unlocked by another process)
	if HasStoredKey() {
		// Try to restore vault state from keyring
		if err := restoreVaultFromKeyring(); err == nil {
			return true
		}
	}

	return false
}

// restoreVaultFromKeyring restores the vault state from system keyring
func restoreVaultFromKeyring() error {
	vaultMutex.Lock()
	defer vaultMutex.Unlock()

	// Double-check after acquiring lock
	if currentVault != nil && currentVault.IsUnlocked {
		return nil
	}

	// Retrieve key from keyring
	symmetricKey, err := RetrieveSymmetricKey()
	if err != nil {
		return err
	}

	// Restore vault state
	currentVault = &VaultState{
		IsUnlocked:   true,
		SymmetricKey: symmetricKey,
	}

	return nil
}

// GetSymmetricKey returns the current symmetric key if vault is unlocked
// Checks both in-memory state and system keyring for cross-process support
func GetSymmetricKey() ([]byte, error) {
	vaultMutex.RLock()
	if currentVault != nil && currentVault.IsUnlocked && currentVault.SymmetricKey != nil {
		// Return a copy to prevent external modification
		keyCopy := make([]byte, len(currentVault.SymmetricKey))
		copy(keyCopy, currentVault.SymmetricKey)
		vaultMutex.RUnlock()
		return keyCopy, nil
	}
	vaultMutex.RUnlock()

	// Try to restore from keyring (unlocked by another process)
	if HasStoredKey() {
		if err := restoreVaultFromKeyring(); err == nil {
			vaultMutex.RLock()
			defer vaultMutex.RUnlock()
			if currentVault != nil && currentVault.SymmetricKey != nil {
				keyCopy := make([]byte, len(currentVault.SymmetricKey))
				copy(keyCopy, currentVault.SymmetricKey)
				return keyCopy, nil
			}
		}
	}

	return nil, fmt.Errorf("vault is locked - run 'ramorie unlock' first")
}

// UnlockVault decrypts the symmetric key with the master password
func UnlockVault(masterPassword string, config *VaultConfig) error {
	vaultMutex.Lock()
	defer vaultMutex.Unlock()

	if !config.EncryptionEnabled {
		return fmt.Errorf("encryption is not enabled for this account")
	}

	// Decode base64 values
	salt, err := Base64ToBytes(config.Salt)
	if err != nil {
		return fmt.Errorf("invalid salt: %w", err)
	}

	encryptedKeyWithNonce, err := Base64ToBytes(config.EncryptedSymmetricKey)
	if err != nil {
		return fmt.Errorf("invalid encrypted key: %w", err)
	}

	// Handle two formats:
	// 1. Frontend format: encrypted_symmetric_key contains [12 bytes nonce][ciphertext]
	// 2. Legacy format: nonce is stored separately in KeyNonce field
	var nonce, encryptedKey []byte
	if config.KeyNonce != "" {
		// Legacy format - nonce is separate
		nonce, err = Base64ToBytes(config.KeyNonce)
		if err != nil {
			return fmt.Errorf("invalid nonce: %w", err)
		}
		encryptedKey = encryptedKeyWithNonce
	} else {
		// Frontend format - nonce is prepended to ciphertext
		// Format: [12 bytes nonce][ciphertext]
		if len(encryptedKeyWithNonce) < NonceLength {
			return fmt.Errorf("encrypted key too short: expected at least %d bytes for nonce", NonceLength)
		}
		nonce = encryptedKeyWithNonce[:NonceLength]
		encryptedKey = encryptedKeyWithNonce[NonceLength:]
	}

	iterations := config.KDFIterations
	if iterations == 0 {
		iterations = PBKDF2Iterations
	}

	// Derive the stretched master key from password
	stretchedMasterKey := DeriveKey(masterPassword, salt, iterations)

	// Decrypt the symmetric key
	symmetricKey, err := DecryptSymmetricKey(encryptedKey, nonce, stretchedMasterKey)
	if err != nil {
		return fmt.Errorf("invalid master password")
	}

	// Store in vault state (in-memory)
	currentVault = &VaultState{
		IsUnlocked:   true,
		SymmetricKey: symmetricKey,
		Salt:         salt,
		Iterations:   iterations,
	}

	// Store symmetric key in system keyring for cross-process access
	if err := StoreSymmetricKey(symmetricKey); err != nil {
		// Log warning but don't fail - in-memory vault still works
		// This might happen on headless systems without keyring
		fmt.Fprintf(os.Stderr, "Warning: Could not store key in system keyring: %v\n", err)
	}

	// Zero out the stretched master key
	for i := range stretchedMasterKey {
		stretchedMasterKey[i] = 0
	}

	return nil
}

// LockVault clears the symmetric key from memory and system keyring
func LockVault() {
	vaultMutex.Lock()
	defer vaultMutex.Unlock()

	if currentVault != nil {
		// Securely zero out the symmetric key
		if currentVault.SymmetricKey != nil {
			for i := range currentVault.SymmetricKey {
				currentVault.SymmetricKey[i] = 0
			}
		}
		currentVault = nil
	}

	// Remove from system keyring
	_ = DeleteSymmetricKey()
}

// GetVaultConfigPath returns the path to the vault config file
func GetVaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ramorie", "vault.json"), nil
}

// LoadVaultConfig loads the vault configuration from disk
func LoadVaultConfig() (*VaultConfig, error) {
	path, err := GetVaultConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No vault config means encryption not set up
			return &VaultConfig{EncryptionEnabled: false}, nil
		}
		return nil, fmt.Errorf("failed to read vault config: %w", err)
	}

	var config VaultConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse vault config: %w", err)
	}

	return &config, nil
}

// SaveVaultConfig saves the vault configuration to disk
func SaveVaultConfig(config *VaultConfig) error {
	path, err := GetVaultConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault config: %w", err)
	}

	// Write with restrictive permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write vault config: %w", err)
	}

	return nil
}

// SetupVaultFromServer updates local vault config from server response
func SetupVaultFromServer(serverConfig map[string]interface{}) error {
	config := &VaultConfig{
		EncryptionEnabled: true,
	}

	if v, ok := serverConfig["encrypted_symmetric_key"].(string); ok {
		config.EncryptedSymmetricKey = v
	}
	if v, ok := serverConfig["key_nonce"].(string); ok {
		config.KeyNonce = v
	}
	if v, ok := serverConfig["salt"].(string); ok {
		config.Salt = v
	}
	if v, ok := serverConfig["kdf_iterations"].(float64); ok {
		config.KDFIterations = int(v)
	}
	if v, ok := serverConfig["kdf_algorithm"].(string); ok {
		config.KDFAlgorithm = v
	}

	return SaveVaultConfig(config)
}

// EncryptContent encrypts content if vault is unlocked, returns original if locked
func EncryptContent(content string) (encrypted, nonce string, isEncrypted bool, err error) {
	key, err := GetSymmetricKey()
	if err != nil {
		// Vault locked - return unencrypted
		return content, "", false, nil
	}

	encrypted, nonce, err = EncryptToBase64(content, key)
	if err != nil {
		return "", "", false, fmt.Errorf("encryption failed: %w", err)
	}

	return encrypted, nonce, true, nil
}

// DecryptContent decrypts content if encrypted and vault is unlocked
func DecryptContent(encryptedContent, nonce string, isEncrypted bool) (string, error) {
	if !isEncrypted {
		return encryptedContent, nil
	}

	key, err := GetSymmetricKey()
	if err != nil {
		return "[Encrypted - Unlock vault to view]", nil
	}

	plaintext, err := DecryptFromBase64(encryptedContent, nonce, key)
	if err != nil {
		return "[Decryption failed]", nil
	}

	return plaintext, nil
}

// --- Organization Encryption ---

// IsOrgVaultUnlocked returns whether a specific org vault is unlocked
func IsOrgVaultUnlocked(orgID string) bool {
	vaultMutex.RLock()
	defer vaultMutex.RUnlock()

	if currentVault == nil || currentVault.OrgKeys == nil {
		return false
	}
	_, ok := currentVault.OrgKeys[orgID]
	return ok
}

// UnlockOrgVault derives the org key from passphrase and stores it in memory
func UnlockOrgVault(orgID, passphrase string, config *OrgVaultConfig) error {
	if !config.IsEnabled {
		return fmt.Errorf("org encryption is not enabled for org %s", orgID)
	}

	salt, err := Base64ToBytes(config.Salt)
	if err != nil {
		return fmt.Errorf("invalid org salt: %w", err)
	}

	iterations := config.KDFIterations
	if iterations == 0 {
		iterations = PBKDF2Iterations
	}

	// Derive org key from passphrase using PBKDF2
	orgKey := DeriveKey(passphrase, salt, iterations)

	vaultMutex.Lock()
	defer vaultMutex.Unlock()

	if currentVault == nil {
		currentVault = &VaultState{}
	}
	if currentVault.OrgKeys == nil {
		currentVault.OrgKeys = make(map[string][]byte)
	}
	currentVault.OrgKeys[orgID] = orgKey

	// Store org key in keyring for cross-process access
	if err := StoreOrgKey(orgID, orgKey); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not store org key in keyring: %v\n", err)
	}

	return nil
}

// LockOrgVault clears a specific org key from memory and keyring
func LockOrgVault(orgID string) {
	vaultMutex.Lock()
	defer vaultMutex.Unlock()

	if currentVault != nil && currentVault.OrgKeys != nil {
		if key, ok := currentVault.OrgKeys[orgID]; ok {
			// Securely zero out the key
			for i := range key {
				key[i] = 0
			}
			delete(currentVault.OrgKeys, orgID)
		}
	}

	// Remove from keyring
	_ = DeleteOrgKey(orgID)
}

// GetOrgSymmetricKey returns the org encryption key for a specific org
func GetOrgSymmetricKey(orgID string) ([]byte, error) {
	vaultMutex.RLock()
	if currentVault != nil && currentVault.OrgKeys != nil {
		if key, ok := currentVault.OrgKeys[orgID]; ok {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			vaultMutex.RUnlock()
			return keyCopy, nil
		}
	}
	vaultMutex.RUnlock()

	// Try to restore from keyring
	orgKey, err := RetrieveOrgKey(orgID)
	if err == nil && orgKey != nil {
		vaultMutex.Lock()
		if currentVault == nil {
			currentVault = &VaultState{}
		}
		if currentVault.OrgKeys == nil {
			currentVault.OrgKeys = make(map[string][]byte)
		}
		currentVault.OrgKeys[orgID] = orgKey
		vaultMutex.Unlock()

		keyCopy := make([]byte, len(orgKey))
		copy(keyCopy, orgKey)
		return keyCopy, nil
	}

	return nil, fmt.Errorf("org vault is locked for org %s - run 'ramorie org unlock' first", orgID)
}

// GetKeyForScope returns the appropriate encryption key based on scope
func GetKeyForScope(scope, orgID string) ([]byte, error) {
	if scope == "organization" && orgID != "" {
		return GetOrgSymmetricKey(orgID)
	}
	return GetSymmetricKey()
}

// EncryptContentWithScope encrypts content using the appropriate key for the scope
func EncryptContentWithScope(content, scope, orgID string) (encrypted, nonce string, isEncrypted bool, err error) {
	key, err := GetKeyForScope(scope, orgID)
	if err != nil {
		// Key not available - return unencrypted
		return content, "", false, nil
	}

	encrypted, nonce, err = EncryptToBase64(content, key)
	if err != nil {
		return "", "", false, fmt.Errorf("encryption failed: %w", err)
	}

	return encrypted, nonce, true, nil
}

// DecryptContentWithScope decrypts content using the appropriate key for the scope
func DecryptContentWithScope(encryptedContent, nonce, scope, orgID string, isEncrypted bool) (string, error) {
	if !isEncrypted {
		return encryptedContent, nil
	}

	key, err := GetKeyForScope(scope, orgID)
	if err != nil {
		if scope == "organization" {
			return "[Org Encrypted - Unlock org vault to view]", nil
		}
		return "[Encrypted - Unlock vault to view]", nil
	}

	plaintext, err := DecryptFromBase64(encryptedContent, nonce, key)
	if err != nil {
		return "[Decryption failed]", nil
	}

	return plaintext, nil
}

// CreateOrgPassphraseHash creates the verification hash for org passphrase
func CreateOrgPassphraseHash(passphrase string, salt []byte, iterations int) (string, error) {
	hash := CreatePasswordHash(passphrase, salt, iterations)
	return hash, nil
}

// RestoreOrgKeyFromKeyring restores an org key from keyring into the vault state
func RestoreOrgKeyFromKeyring(orgID string, orgKey []byte) {
	vaultMutex.Lock()
	defer vaultMutex.Unlock()

	if currentVault == nil {
		currentVault = &VaultState{}
	}
	if currentVault.OrgKeys == nil {
		currentVault.OrgKeys = make(map[string][]byte)
	}
	currentVault.OrgKeys[orgID] = orgKey
}
