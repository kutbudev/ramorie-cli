// Package encstate resolves the EFFECTIVE personal-encryption decision used by
// every write path (CLI commands + MCP tools).
//
// Historically the encrypt-on-write decision trusted the LOCAL cached
// `config.EncryptionEnabled` flag (only ever populated at `ramorie setup` time)
// together with `crypto.IsVaultUnlocked()` (which reads the OS keyring that
// persists across processes). When a user disabled encryption in the web app,
// neither of those local signals changed — so the CLI/MCP kept writing
// encrypted content with the OLD personal key the server could no longer
// unwrap, and the web showed "[Encrypted by another user]".
//
// This package makes the decision reflect the SERVER's CURRENT
// `encryption_enabled` status, with a short per-process TTL cache to avoid an
// API round-trip on every write, graceful degradation on fetch failure, and a
// self-heal that locks the local vault the moment the server confirms
// encryption is disabled.
package encstate

import (
	"sync"
	"time"

	"github.com/kutbudev/ramorie-cli/internal/config"
	"github.com/kutbudev/ramorie-cli/internal/crypto"
)

// cacheTTL is how long a successfully fetched server status is trusted before
// we re-fetch. Short enough that disabling encryption in the web app stops
// CLI/MCP encryption within ~1 minute, long enough to avoid hammering the API
// on bursts of writes.
const cacheTTL = 60 * time.Second

// StatusFetcher fetches the user's current server-side encryption status.
// *api.Client satisfies this via GetEncryptionConfig (see ServerStatus wiring
// in encstate_api.go). It is an interface so the resolver is testable without
// network.
type StatusFetcher interface {
	// FetchEncryptionEnabled returns the server's current encryption_enabled
	// flag. A non-nil error signals the status could not be determined.
	FetchEncryptionEnabled() (bool, error)
}

var (
	mu          sync.Mutex
	cached      bool
	cachedValid bool
	cachedAt    time.Time
)

// nowFunc is overridable in tests.
var nowFunc = time.Now

// resetForTest clears the per-process cache. Test-only.
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	cached = false
	cachedValid = false
	cachedAt = time.Time{}
}

// ShouldEncryptPersonal reports whether a personal-scope write should be
// encrypted, based on the SERVER's current encryption_enabled status.
//
// Behaviour:
//   - Fetch succeeds, server ENABLED  -> returns true  (cache the result).
//   - Fetch succeeds, server DISABLED -> returns false (cache the result) AND
//     self-heals: persists cfg.EncryptionEnabled=false and locks the vault so
//     the keyring key is wiped and subsequent writes are plaintext immediately.
//   - Fetch fails (network/server)    -> graceful fallback to the locally
//     cached cfg.EncryptionEnabled; NEVER auto-locks on a fetch error.
//
// Note: this returns the encryption-ENABLED decision only. Callers must still
// require crypto.IsVaultUnlocked() (and !isOrgProject) before actually
// encrypting — this helper deliberately does not check vault state so it can
// drive the self-heal independently.
func ShouldEncryptPersonal(fetcher StatusFetcher) bool {
	// Serve from cache while fresh.
	mu.Lock()
	if cachedValid && nowFunc().Sub(cachedAt) < cacheTTL {
		v := cached
		mu.Unlock()
		return v
	}
	mu.Unlock()

	// Local fallback value used on fetch failure.
	localEnabled := false
	if cfg, err := config.LoadConfig(); err == nil && cfg != nil {
		localEnabled = cfg.EncryptionEnabled
	}

	if fetcher == nil {
		// Cannot consult the server — degrade to local state, do not cache.
		return localEnabled
	}

	serverEnabled, err := fetcher.FetchEncryptionEnabled()
	if err != nil {
		// Graceful degradation: trust the last-known local flag, never
		// auto-lock on a transient fetch error.
		return localEnabled
	}

	// Confirmed server response — cache it.
	mu.Lock()
	cached = serverEnabled
	cachedValid = true
	cachedAt = nowFunc()
	mu.Unlock()

	if !serverEnabled {
		// Self-heal: server confirms encryption is OFF. Stop encrypting now and
		// for every future write this process makes, and clear the stale key so
		// other processes (and the next process) also stop.
		selfHealDisabled()
	}

	return serverEnabled
}

// selfHealDisabled is invoked only on a CONFIRMED server "disabled" response.
// It persists the local config flag as false and locks the personal vault
// (which also deletes the symmetric key from the OS keyring). Errors are
// swallowed: self-heal is best-effort and must never break a write.
func selfHealDisabled() {
	if cfg, err := config.LoadConfig(); err == nil && cfg != nil && cfg.EncryptionEnabled {
		cfg.EncryptionEnabled = false
		_ = config.SaveConfig(cfg)
	}
	// Lock unconditionally on confirmed-disabled: clears in-memory key + keyring
	// so IsVaultUnlocked() stops returning true across processes.
	if crypto.IsVaultUnlocked() {
		crypto.LockVault()
	}
}
