package blevestore

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

import (
	"github.com/blevesearch/bleve/index/store"
	"github.com/dgraph-io/badger"
)

type Reader struct {
	store         *Store
	txn           *badger.Txn
	indexPrefixID []byte
}

func (r *Reader) Get(key []byte) ([]byte, error) {
	var rv []byte

	item, err := r.txn.Get(r.store.buildID(key))
	if err != nil {
		return nil, err
	}

	_, err = item.ValueCopy(rv)
	return rv, err
}

func (r *Reader) MultiGet(keys [][]byte) ([][]byte, error) {
	rvs := make([][]byte, len(keys))

	for i, key := range keys {
		item, err := r.txn.Get(r.store.buildID(key))
		if err != nil {
			return nil, err
		}

		_, err = item.ValueCopy(rvs[i])
		if err != nil {
			return nil, err
		}
	}

	return rvs, nil
}

func (r *Reader) iterator() *Iterator {
	txn := r.store.db.NewTransaction(false)
	iter := txn.NewIterator(badger.DefaultIteratorOptions)

	rv := &Iterator{
		store:    r.store,
		txn:      txn,
		iterator: iter,
	}

	return rv
}
func (r *Reader) PrefixIterator(prefix []byte) store.KVIterator {
	rv := r.iterator()
	rv.prefix = prefix

	rv.Seek(r.store.buildID(prefix))
	return rv
}

func (r *Reader) RangeIterator(start, end []byte) store.KVIterator {
	rv := r.iterator()
	rv.start = start
	rv.end = end

	rv.Seek(r.store.buildID(start))
	return rv
}

func (r *Reader) Close() error {
	r.txn.Discard()
	return nil
}
