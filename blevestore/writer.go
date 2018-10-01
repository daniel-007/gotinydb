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

	"github.com/alexandrestein/gotinydb/cipher"
	"github.com/blevesearch/bleve/index/store"
	"github.com/dgraph-io/badger"
)

type Writer struct {
	store *Store
}

func (w *Writer) NewBatch() store.KVBatch {
	return store.NewEmulatedBatch(w.store.mo)
}

func (w *Writer) set(dbID, content []byte) error {
	req := NewBleveStoreWriteRequest(dbID, content)

	fmt.Println("writer send")
	w.store.config.bleveWriteChan <- req

	fmt.Println("writer wait err")
	err := <-req.ResponseChan

	return err
}

func (w *Writer) NewBatchEx(options store.KVBatchOptions) ([]byte, store.KVBatch, error) {
	return make([]byte, options.TotalBytes), w.NewBatch(), nil
}

func (w *Writer) ExecuteBatch(batch store.KVBatch) (err error) {
	emulatedBatch, ok := batch.(*store.EmulatedBatch)
	if !ok {
		return fmt.Errorf("wrong type of batch")
	}

	// txn := w.store.db.NewTransaction(true)
	// localTxn := false
	// txn := w.store.config.writeTxn
	// if txn == nil {
	txn := w.store.config.db.NewTransaction(false)
	defer txn.Discard()
	// 	localTxn = true
	// }

	// if localTxn {
	// 	// defer function to ensure that once started,
	// 	// we either Commit tx or Rollback
	// 	defer func() {
	// 		// if nothing went wrong, commit
	// 		if err == nil {
	// 			// careful to catch error here too
	// 			err = txn.Commit(nil)
	// 		}
	// 	}()
	// }

	for k, mergeOps := range emulatedBatch.Merger.Merges {
		kb := []byte(k)

		storeID := w.store.buildID(kb)

		var item *badger.Item
		existingVal := []byte{}
		item, err = txn.Get(storeID)
		// If the KV pair exists the existing value is saved
		if err == nil {
			var encryptedValue []byte
			encryptedValue, err = item.ValueCopy(existingVal)
			if err != nil {
				return
			}

			existingVal, err = cipher.Decrypt(w.store.config.key, storeID, encryptedValue)
			if err != nil {
				return
			}
		}

		mergedVal, fullMergeOk := w.store.mo.FullMerge(kb, existingVal, mergeOps)
		if !fullMergeOk {
			err = fmt.Errorf("merge operator returned failure")
			return
		}

		// err = txn.Set(storeID, cipher.Encrypt(w.store.config.key, storeID, mergedVal))
		err = w.set(storeID, cipher.Encrypt(w.store.config.key, storeID, mergedVal))
		if err != nil {
			return
		}
	}

	for _, op := range emulatedBatch.Ops {
		storeID := w.store.buildID(op.K)

		if op.V != nil {
			// err = txn.Set(storeID, cipher.Encrypt(w.store.config.key, storeID, op.V))
			err = w.set(storeID, cipher.Encrypt(w.store.config.key, storeID, op.V))
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
