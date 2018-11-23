package securecache

import (
	"time"

	"github.com/muesli/cache2go"
)

var ()

// Those defines the cache tables constant values
var (
	WaitingRequestTable = cache2go.Cache("waiting requests")
	// PeersTable          = cache2go.Cache("peers")

	CacheValueWaitingRequestsTimeOut = time.Minute * 10
)
