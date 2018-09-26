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
	"fmt"

	"github.com/blevesearch/bleve/index/store"
	"github.com/dgraph-io/badger"

	"github.com/alexandrestein/gotinydb/cipher"
)

type Writer struct {
	store *Store
}

func (w *Writer) NewBatch() store.KVBatch {
	return store.NewEmulatedBatch(w.store.mo)
}

func (w *Writer) NewBatchEx(options store.KVBatchOptions) ([]byte, store.KVBatch, error) {
	return make([]byte, options.TotalBytes), w.NewBatch(), nil
}

func (w *Writer) ExecuteBatch(batch store.KVBatch) (err error) {
	emulatedBatch, ok := batch.(*store.EmulatedBatch)
	if !ok {
		return fmt.Errorf("wrong type of batch")
	}

	txn := w.store.db.NewTransaction(true)
	// defer function to ensure that once started,
	// we either Commit tx or Rollback
	defer func() {
		// if nothing went wrong, commit
		if err == nil {
			// careful to catch error here too
			err = txn.Commit(nil)
		}
		txn.Discard()
	}()

	for k, mergeOps := range emulatedBatch.Merger.Merges {
		kb := []byte(k)

		var item *badger.Item
		existingVal := []byte{}
		item, err = txn.Get(w.store.buildID(kb))
		// If the KV pair exists the existing value is saved
		if err == nil {
			existingVal, err = item.ValueCopy(existingVal)
			if err != nil {
				return
			}
		}

		mergedVal, fullMergeOk := w.store.mo.FullMerge(kb, existingVal, mergeOps)
		if !fullMergeOk {
			err = fmt.Errorf("merge operator returned failure")
			return
		}

		storeID := w.store.buildID(kb)
		err = txn.Set(storeID, cipher.Encrypt(w.store.primaryEncryptionKey, storeID, mergedVal))
		if err != nil {
			return
		}
	}

	for _, op := range emulatedBatch.Ops {
		storeID := w.store.buildID(op.K)

		if op.V != nil {
			err = txn.Set(storeID, cipher.Encrypt(w.store.primaryEncryptionKey, storeID, op.V))
			if err != nil {
				return
			}
		} else {
			err = txn.Delete(storeID)
			if err != nil {
				return
			}
		}
	}
	return
}

func (w *Writer) Close() error {
	return nil
}