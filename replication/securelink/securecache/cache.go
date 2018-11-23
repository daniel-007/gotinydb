package securecache

import (
	"time"

	"github.com/muesli/cache2go"
)

var (
	WaitingRequestTable = cache2go.Cache("waiting requests")
	// PeersTable          = cache2go.Cache("peers")
)

// Those defines the cache tables constant values
var (
	CacheValueWaitingRequestsTimeOut = time.Minute * 10
)
