package credentials

import (
	"context"
	"sync"
	"time"
)

const keychainWriteCheckTTL = 5 * time.Minute

var (
	writeCheckMu     sync.Mutex
	writeCheckAt     time.Time
	writeCheckOK     bool
	writeCheckDetail string
)

// CachedKeychainWriteAvailable probes keychain write once per process (refreshed periodically).
func CachedKeychainWriteAvailable(ctx context.Context) (ok bool, detail string) {
	writeCheckMu.Lock()
	defer writeCheckMu.Unlock()
	if !writeCheckAt.IsZero() && time.Since(writeCheckAt) < keychainWriteCheckTTL {
		return writeCheckOK, writeCheckDetail
	}
	ok, detail = KeychainWriteAvailable(ctx)
	writeCheckOK, writeCheckDetail, writeCheckAt = ok, detail, time.Now()
	return ok, detail
}

// ResetKeychainWriteCache clears the write-probe cache (tests).
func ResetKeychainWriteCache() {
	writeCheckMu.Lock()
	writeCheckAt = time.Time{}
	writeCheckMu.Unlock()
}