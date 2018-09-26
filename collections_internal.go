package gotinydb

import (
	"context"

	"github.com/alexandrestein/gotinydb/cipher"
	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
)

func (c *Collection) buildCollectionPrefix() []byte {
	return []byte{c.prefix}
}

func (c *Collection) buildIDWhitPrefixData(id []byte) []byte {
	ret := []byte{c.prefix, prefixData}
	return append(ret, id...)
}

// func (c *Collection) buildIDWhitPrefixIndex(indexName, id []byte) []byte {
// 	ret := []byte{c.prefix, prefixIndexes}

// 	bs := blake2b.Sum256(indexName)

// 	ret = append(ret, bs[:8]...)
// 	ret = append(ret, indexName...)
// 	return append(ret, id...)
// }

func (c *Collection) buildStoreID(id string) []byte {
	return c.buildIDWhitPrefixData([]byte(id))
}

func (c *Collection) putIntoIndexes(id string, data interface{}) error {
	for _, i := range c.indexes {
		err := i.index.Index(id, data)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Collection) insertOrDeleteStore(ctx context.Context, txn *badger.Txn, isInsertion bool, writeTransaction *writeTransactionElement) error {

	storeID := c.buildStoreID(writeTransaction.id)

	if isInsertion {
		e := &badger.Entry{
			Key:   storeID,
			Value: cipher.Encrypt(c.options.privateCryptoKey, storeID, writeTransaction.contentAsBytes),
		}

		return txn.SetEntry(e)
	}
	return txn.Delete(storeID)
}

func (c *Collection) get(ctx context.Context, ids ...string) ([][]byte, error) {
	ret := make([][]byte, len(ids))
	return ret, c.store.View(func(txn *badger.Txn) error {
		for i, id := range ids {
			idAsBytes := c.buildStoreID(id)
			item, err := txn.Get(idAsBytes)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					return ErrNotFound
				}
				return err
			}

			if item.IsDeletedOrExpired() {
				return ErrNotFound
			}

			var contentAsEncryptedBytes []byte
			contentAsEncryptedBytes, err = item.ValueCopy(contentAsEncryptedBytes)
			if err != nil {
				return err
			}

			var contentAsBytes []byte
			contentAsBytes, err = cipher.Decrypt(c.options.privateCryptoKey, item.Key(), contentAsEncryptedBytes)
			if err != nil {
				return err
			}

			ret[i] = contentAsBytes
		}
		return nil
	})
}

// getStoredIDs returns all ids if it does not exceed the limit.
// This will not returned the ID used to set the value inside the collection
// It returns the id used to set the value inside the store
func (c *Collection) getStoredIDsAndValues(starter string, limit int, IDsOnly bool) ([]*ResponseElem, error) {
	response := make([]*ResponseElem, limit)

	return response, c.store.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.DefaultIteratorOptions)
		defer iter.Close()

		prefix := c.buildIDWhitPrefixData(nil)
		iter.Seek(c.buildIDWhitPrefixData([]byte(starter)))

		count := 0
		for ; iter.Valid(); iter.Next() {
			if !iter.ValidForPrefix(prefix) || count > limit-1 {
				response = response[:count]
				return nil
			}

			responseItem := new(ResponseElem)

			item := iter.Item()

			if item.IsDeletedOrExpired() {
				continue
			}

			responseItem._ID = new(idType)
			responseItem._ID.ID = string(item.Key()[len(c.buildIDWhitPrefixData(nil)):])

			if !IDsOnly {
				var err error
				responseItem.contentAsBytes, err = item.ValueCopy(responseItem.contentAsBytes)
				if err != nil {
					return err
				}

				responseItem.contentAsBytes, err = cipher.Decrypt(c.options.privateCryptoKey, item.Key(), responseItem.contentAsBytes)
				if err != nil {
					return err
				}
			}

			response[count] = responseItem

			count++
		}

		// Clean the end of the slice if not full
		response = response[:count]
		return nil
	})
}

func (c *Collection) indexAllValues() error {
	lastID := ""

newLoop:
	savedElements, getErr := c.getStoredIDsAndValues(lastID, c.options.PutBufferLimit, false)
	if getErr != nil {
		return getErr
	}

	if len(savedElements) <= 1 {
		return nil
	}

	txn := c.store.NewTransaction(true)
	defer txn.Discard()

	for _, savedElement := range savedElements {
		if savedElement.GetID() == lastID {
			continue
		}

		err := c.putIntoIndexes(savedElement.GetID(), savedElement.contentAsBytes)
		if err != nil {
			return err
		}

		lastID = savedElement.GetID()
	}

	err := txn.Commit(nil)
	if err != nil {
		return err
	}

	goto newLoop
}

func (c *Collection) isRunning() bool {
	if c.ctx.Err() != nil {
		return false
	}

	return true
}

func (c *Collection) buildKvConfig(indexPrefix byte) map[string]interface{} {
	collectionAndIndexPrefix := []byte{c.prefix, indexPrefix}
	return map[string]interface{}{"path": "test", "prefix": collectionAndIndexPrefix, "db": c.store, "key": c.options.privateCryptoKey}
}

func (c *Collection) getIndex(name string) (*index, error) {
	var index *index

	// Loop all indexes to found the given index
	found := false
	for _, i := range c.indexes {
		if i.Name == name {
			index = i
			found = true
			break
		}
	}

	if !found {
		return nil, ErrIndexNotFound
	}

	// If index is already loaded
	if index.index != nil {
		return index, nil
	}

	// Load the index
	bleveIndex, err := bleve.OpenUsing(c.options.Path+"/"+c.name+"/"+index.Name, c.buildKvConfig(index.Prefix))
	if err != nil {
		return nil, err
	}

	// Save the index interface into the internal index type
	index.index = bleveIndex
	index.collectionPrefix = c.prefix

	return index, nil
}

