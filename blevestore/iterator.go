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
	"bytes"

	"github.com/dgraph-io/badger"

	"github.com/alexandrestein/gotinydb/cipher"
)

type Iterator struct {
	store    *Store
	iterator *badger.Iterator
	prefix   []byte
	start    []byte
	end      []byte
	// valid    bool
	// key      []byte
	// val      []byte
}

// func (i *Iterator) updateValid() {
// 	i.valid = (i.key != nil)
// 	if i.valid {
// 		if i.prefix != nil {
// 			i.valid = bytes.HasPrefix(i.key, i.prefix)
// 		} else if i.end != nil {
// 			i.valid = bytes.Compare(i.key, i.end) < 0
// 		}
// 	}
// }

func (i *Iterator) Seek(k []byte) {
	if i.start != nil && bytes.Compare(k, i.start) < 0 {
		k = i.start
	}
	if i.prefix != nil && !bytes.HasPrefix(k, i.prefix) {
		if bytes.Compare(k, i.prefix) < 0 {
			k = i.prefix
			// } else {
			// 	i.valid = false
			// 	return
		}
	}
	// i.key, i.val = i.cursor.Seek(k)
	// i.updateValid()

	i.iterator.Seek(i.store.buildID(k))
}

func (i *Iterator) Next() {
	i.iterator.Next()
}

func (i *Iterator) Current() (key []byte, val []byte, valid bool) {
	valid = i.iterator.ValidForPrefix(i.store.buildID(key))
	if !valid {
		return
	}

	key = i.Key()
	val = i.Value()

	return
}

func (i *Iterator) key() (key []byte) {
	key = []byte{}
	key = i.iterator.Item().KeyCopy(key)
	key = key[i.store.indexPrefixIDLen:]

	return
}

func (i *Iterator) Key() (key []byte) {
	if !i.Valid() {
		return
	}
	return i.key()
}

func (i *Iterator) Value() (val []byte) {
	if !i.Valid() {
		return
	}

	item := i.iterator.Item()

	var encryptVal []byte
	encryptVal, _ = item.ValueCopy(encryptVal)

	val, _ = cipher.Decrypt(i.store.primaryEncryptionKey, item.Key(), encryptVal)

	return
}

func (i *Iterator) Valid() bool {
	if !i.iterator.Valid() {
		return false
	}
	if i.prefix != nil {
		return i.iterator.ValidForPrefix(i.store.buildID(i.prefix))
	} else if i.end != nil {
		return bytes.Compare(i.key(), i.end) < 0
	}
	return i.iterator.ValidForPrefix(i.store.buildID(nil))
}

func (i *Iterator) Close() error {
	i.iterator.Close()
	return nil
}