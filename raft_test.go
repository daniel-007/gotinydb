package gotinydb

import (
	"math/rand"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/raft"
)

func TestRaftStores(t *testing.T) {
	db, err := Open(os.TempDir()+"/testDB", [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(os.TempDir() + "/testDB")

	raftStore := &raftStore{db}

	t.Run("stable", func(t *testing.T) {
		testStableStore(t, raftStore)
	})
	t.Run("log", func(t *testing.T) {
		testLogStore(t, raftStore)
	})
	t.Run("snapshot", func(t *testing.T) {
		testSnapshotStore(t, db, raftStore)
	})
}

func testStableStore(t *testing.T, rs *raftStore) {
	key := []byte("key")
	val := []byte("val")

	_, err := rs.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rs.GetUint64(key)
	if err != nil {
		t.Fatal(err)
	}
	err = nil

	err = rs.Set(key, val)
	if err != nil {
		t.Fatal(err)
	}

	var getVal []byte
	getVal, err = rs.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(val, getVal) {
		t.Fatalf("the values are not equal but should:\n\t%v\n\t%v", val, getVal)
	}

	uintKey := []byte("uintKey")
	uintVal := rand.Uint64()
	err = rs.SetUint64(uintKey, uintVal)
	if err != nil {
		t.Fatal(err)
	}

	var getUintVal uint64
	getUintVal, err = rs.GetUint64(uintKey)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(uintVal, getUintVal) {
		t.Fatalf("the values are not equal but should:\n\t%v\n\t%v", uintVal, getUintVal)
	}
}

func testLogStore(t *testing.T, rs *raftStore) {
	testLogStoreFirstAndLastOnEmpty(t, rs)

	log1 := &raft.Log{
		Index: 1,
		Term:  rand.Uint64(),
		Type:  raft.LogCommand,
		Data:  []byte{},
	}
	log2 := &raft.Log{
		Index: rand.Uint64(),
		Term:  rand.Uint64(),
		Type:  raft.LogCommand,
		Data:  []byte{},
	}

	if err := rs.StoreLog(log1); err != nil {
		t.Fatal(err)
	}
	if err := rs.StoreLog(log2); err != nil {
		t.Fatal(err)
	}

	testLogStoreFirstAndLast(t, rs, log1.Index, log2.Index)

	retrievedLog1 := new(raft.Log)
	if err := rs.GetLog(log1.Index, retrievedLog1); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(log1, retrievedLog1) {
		t.Fatalf("the logs must be equal but not:\n\t%v\n\t%v\n%v", log1, retrievedLog1, reflect.DeepEqual(log1, retrievedLog1))
	}

	if err := rs.DeleteRange(log1.Index, log2.Index); err != nil {
		t.Fatal(err)
	}

	if err := rs.GetLog(log1.Index, nil); err == nil {
		t.Fatalf("no error but the element was deleted")
	}
	if err := rs.GetLog(log2.Index, nil); err == nil {
		t.Fatalf("no error but the element was deleted")
	}

	testLogStoreFirstAndLastOnEmpty(t, rs)

	for i := 0; i < 3; i++ {
		tmpLogs := make([]*raft.Log, 100)
		for j := 0; j < 100; j++ {
			tmpLog := &raft.Log{
				Index: uint64(i*100 + j),
				Term:  rand.Uint64(),
				Type:  raft.LogCommand,
				Data:  []byte{},
			}
			tmpLogs[j] = tmpLog
		}
		err := rs.StoreLogs(tmpLogs)
		if err != nil {
			t.Fatal(err)
		}
	}

	testLogStoreFirstAndLast(t, rs, 0, 299)
}

func testLogStoreFirstAndLastOnEmpty(t *testing.T, rs *raftStore) {
	if i, err := rs.FirstIndex(); i != 0 {
		t.Fatalf("the expected index is 0 but had: %d", i)
	} else if err == nil {
		t.Fatalf("it should return an error but not")
	}
	if i, err := rs.LastIndex(); i != 0 {
		t.Fatalf("the expected index is 0 but had: %d", i)
	} else if err == nil {
		t.Fatalf("it should return an error but not")
	}
}

func testLogStoreFirstAndLast(t *testing.T, rs *raftStore, f, l uint64) {
	if i, err := rs.FirstIndex(); i != f {
		t.Fatalf("the expected index is 1 but had: %d", i)
	} else if err != nil {
		t.Fatal(err)
	} else {
		if testing.Verbose() {
			t.Logf("the first index is %d", i)
		}
	}

	if i, err := rs.LastIndex(); i != l {
		t.Fatalf("the expected index is %d but had: %d", l, i)
	} else if err != nil {
		t.Fatal(err)
	} else {
		if testing.Verbose() {
			t.Logf("the last index is %d", i)
		}
	}
}

func testSnapshotStore(t *testing.T, db *DB, rs *raftStore) {
	t.SkipNow()
	// ca, _ := securelink.NewCA(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "ca")
	// cert, _ := ca.NewCert(securelink.KeyTypeEc, securelink.KeyLengthEc256, time.Hour, securelink.GetCertTemplate(nil, nil), "node")

	// addr, err := common.NewAddr(1323)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// t.Log(addr.Addrs)

	// node, err := replication.NewNode(addr, rs, cert)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// tr := node.GetRaftTransport()

	// rs.Create(raft.SnapshotVersionMax, 1, 1, )
}
