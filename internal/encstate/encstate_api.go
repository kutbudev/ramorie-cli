package encstate

import "github.com/kutbudev/ramorie-cli/internal/api"

// clientFetcher adapts *api.Client to the StatusFetcher interface by calling
// the existing GetEncryptionConfig endpoint (GET /auth/encryption-status).
type clientFetcher struct {
	client *api.Client
}

// FetchEncryptionEnabled implements StatusFetcher.
func (f clientFetcher) FetchEncryptionEnabled() (bool, error) {
	cfg, err := f.client.GetEncryptionConfig()
	if err != nil {
		return false, err
	}
	if cfg == nil {
		return false, errNilConfig
	}
	return cfg.EncryptionEnabled, nil
}

// errNilConfig is returned when the server response unmarshals to nil; treated
// as a fetch error so callers fall back to local state rather than self-healing.
var errNilConfig = errNil("encryption config is nil")

type errNil string

func (e errNil) Error() string { return string(e) }

// FetcherFor returns a StatusFetcher backed by the given API client, or nil if
// the client is nil (callers degrade to local state).
func FetcherFor(client *api.Client) StatusFetcher {
	if client == nil {
		return nil
	}
	return clientFetcher{client: client}
}
