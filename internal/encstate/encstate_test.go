package encstate

import (
	"errors"
	"testing"
)

// fakeFetcher is a table-driven StatusFetcher: returns the configured value or
// error and counts calls so we can assert caching behaviour.
type fakeFetcher struct {
	enabled bool
	err     error
	calls   int
}

func (f *fakeFetcher) FetchEncryptionEnabled() (bool, error) {
	f.calls++
	return f.enabled, f.err
}

func TestShouldEncryptPersonal(t *testing.T) {
	tests := []struct {
		name       string
		fetcher    *fakeFetcher
		nilFetcher bool
		want       bool
	}{
		{
			name:    "server enabled -> encrypt",
			fetcher: &fakeFetcher{enabled: true},
			want:    true,
		},
		{
			name:    "server disabled -> do not encrypt",
			fetcher: &fakeFetcher{enabled: false},
			want:    false,
		},
		{
			name:    "fetch error -> fall back to local (no config in test -> false)",
			fetcher: &fakeFetcher{err: errors.New("network down")},
			want:    false,
		},
		{
			name:       "nil fetcher -> fall back to local (false)",
			nilFetcher: true,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetForTest()
			var f StatusFetcher
			if !tt.nilFetcher {
				f = tt.fetcher
			}
			got := ShouldEncryptPersonal(f)
			if got != tt.want {
				t.Errorf("ShouldEncryptPersonal() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestShouldEncryptPersonal_CachesSuccess verifies a confirmed server response
// is cached so repeated writes within the TTL don't hit the API again.
func TestShouldEncryptPersonal_CachesSuccess(t *testing.T) {
	resetForTest()
	f := &fakeFetcher{enabled: true}

	for i := 0; i < 5; i++ {
		if !ShouldEncryptPersonal(f) {
			t.Fatalf("call %d: expected true", i)
		}
	}
	if f.calls != 1 {
		t.Errorf("expected fetcher called once (cached), got %d calls", f.calls)
	}
}

// TestShouldEncryptPersonal_ErrorNotCached verifies a fetch error is NOT cached:
// the next call retries the fetcher (so recovery is picked up promptly).
func TestShouldEncryptPersonal_ErrorNotCached(t *testing.T) {
	resetForTest()
	f := &fakeFetcher{err: errors.New("boom")}

	_ = ShouldEncryptPersonal(f)
	_ = ShouldEncryptPersonal(f)

	if f.calls != 2 {
		t.Errorf("expected fetcher called twice (error not cached), got %d calls", f.calls)
	}
}
