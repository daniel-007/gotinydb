package securecache

import (
	"fmt"
	"time"

	"github.com/coreos/etcd/raft"

	"github.com/muesli/cache2go"
)

var (
	WaitingRequestTable = cache2go.Cache("waiting requests")
	PeersTable          = cache2go.Cache("peers")
)

// Those defines the cache tables constant values
var (
	CacheValueWaitingRequestsTimeOut = time.Minute * 10
)

type (
	SavedPeer struct {
		Addrs []string
		Port  string
	}
)

func GetPeers() []raft.Peer {
	ret := []raft.Peer{}
	PeersTable.Foreach(func(key interface{}, item *cache2go.CacheItem) {
		keyAsInt64 := key.(int64)
		peer := raft.Peer{
			ID: uint64(keyAsInt64),
		}
		ret = append(ret, peer)

		fmt.Println("peer", keyAsInt64, item.Data().(*SavedPeer).Port)
	})

	return ret
}
