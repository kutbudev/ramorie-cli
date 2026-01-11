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
func IsVaultUnlocked() bool {
	vaultMutex.RLock()
	defer vaultMutex.RUnlock()
	return currentVault != nil && currentVault.IsUnlocked
}

// GetSymmetricKey returns the current symmetric key if vault is unlocked
func GetSymmetricKey() ([]byte, error) {
	vaultMutex.RLock()
	defer vaultMutex.RUnlock()

	if currentVault == nil || !currentVault.IsUnlocked {
		return nil, fmt.Errorf("vault is locked - run 'ramorie unlock' first")
	}

	if currentVault.SymmetricKey == nil {
		return nil, fmt.Errorf("symmetric key not available")
	}

	// Return a copy to prevent external modification
	keyCopy := make([]byte, len(currentVault.SymmetricKey))
	copy(keyCopy, currentVault.SymmetricKey)
	return keyCopy, nil
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

	encryptedKey, err := Base64ToBytes(config.EncryptedSymmetricKey)
	if err != nil {
		return fmt.Errorf("invalid encrypted key: %w", err)
	}

	nonce, err := Base64ToBytes(config.KeyNonce)
	if err != nil {
		return fmt.Errorf("invalid nonce: %w", err)
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

	// Store in vault state
	currentVault = &VaultState{
		IsUnlocked:   true,
		SymmetricKey: symmetricKey,
		Salt:         salt,
		Iterations:   iterations,
	}

	// Zero out the stretched master key
	for i := range stretchedMasterKey {
		stretchedMasterKey[i] = 0
	}

	return nil
}

// LockVault clears the symmetric key from memory
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
