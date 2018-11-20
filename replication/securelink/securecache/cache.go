package securecache

import (
	"time"

	"github.com/muesli/cache2go"
)

var (
	WaitingRequestTable = cache2go.Cache("waiting requests")
)

// Those defines the cache tables constant values
var (
	CacheValueWaitingRequestsTimeOut = time.Minute * 10
)

type (
	cache cache2go.CacheTable
)
