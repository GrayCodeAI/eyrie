package discover

import "sync"

// runMu serializes catalog discovery so concurrent refresh calls do not corrupt cache writes.
var runMu sync.Mutex
