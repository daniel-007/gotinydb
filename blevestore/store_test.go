//  Copyright (c) 2014 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blevestore

import (
	"os"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger"

	"github.com/blevesearch/bleve/index/store"
	"github.com/blevesearch/bleve/index/store/test"
)

func open(t *testing.T, mo store.MergeOperator) store.KVStore {
	opt := badger.DefaultOptions
	opt.Dir = "test"
	opt.ValueDir = "test"
	db, err := badger.Open(opt)
	if err != nil {
		t.Fatal(err)
	}
	var rv store.KVStore
	rv, err = New(mo, map[string]interface{}{"path": "test", "prefix": []byte{1, 9}, "db": db, "key": [32]byte{}})
	if err != nil {
		t.Fatal(err)
	}
	return rv
}

func cleanup(t *testing.T, s store.KVStore) {
	err := s.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = os.RemoveAll("test")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBoltDBKVCrud(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestKVCrud(t, s)
}

func TestBoltDBReaderIsolation(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestReaderIsolation(t, s)
}

func TestBoltDBReaderOwnsGetBytes(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestReaderOwnsGetBytes(t, s)
}

func TestBoltDBWriterOwnsBytes(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestWriterOwnsBytes(t, s)
}

func TestBoltDBPrefixIterator(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestPrefixIterator(t, s)
}

func TestBoltDBPrefixIteratorSeek(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestPrefixIteratorSeek(t, s)
}

func TestBoltDBRangeIterator(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestRangeIterator(t, s)
}

func TestBoltDBRangeIteratorSeek(t *testing.T) {
	s := open(t, nil)
	defer cleanup(t, s)
	test.CommonTestRangeIteratorSeek(t, s)
}

func TestBoltDBMerge(t *testing.T) {
	s := open(t, &test.TestMergeCounter{})
	defer cleanup(t, s)
	test.CommonTestMerge(t, s)
}

func TestBoltDBConfig(t *testing.T) {
	path := "test"
	defer os.RemoveAll(path)
	os.RemoveAll(path)

	opt := badger.DefaultOptions
	opt.Dir = path
	db, _ := badger.Open(opt)

	var tests = []struct {
		in                   map[string]interface{}
		name                 string
		primaryEncryptionKey [32]byte
		indexPrefixID        []byte
		db                   *badger.DB
	}{
		{
			map[string]interface{}{"path": "test", "prefix": []byte{0}},
			"test",
			[32]byte{},
			[]byte{2, 5},
			db,
		},
		{
			map[string]interface{}{"path": "test"},
			"test 2",
			[32]byte{},
			[]byte{2, 5},
			db,
		},
	}

	for _, test := range tests {
		kv, err := New(nil, test.in)
		if err != nil {
			t.Fatal(err)
		}
		bs, ok := kv.(*Store)
		if !ok {
			t.Fatal("failed type assertion to *boltdb.Store")
		}
		if bs.name != test.name {
			t.Fatalf("path: expected %q, got %q", test.name, bs.name)
		}
		if reflect.DeepEqual(bs.indexPrefixID, test.indexPrefixID) {
			t.Fatalf("prefix: expected %q, got %q", test.indexPrefixID, bs.indexPrefixID)
		}
		if bs.db != test.db {
			t.Fatalf("db: expected %v, got %v", test.db, bs.db)
		}
		if bs.primaryEncryptionKey != test.primaryEncryptionKey {
			t.Fatalf("key: expected %v, got %v", test.primaryEncryptionKey, bs.primaryEncryptionKey)
		}
	}
}
