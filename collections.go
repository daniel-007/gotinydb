package gotinydb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/alexandrestein/gotinydb/blevestore"
	"github.com/alexandrestein/gotinydb/cipher"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/index/upsidedown"
	"github.com/blevesearch/bleve/mapping"
	"github.com/dgraph-io/badger"
)

// Put add the given content to database with the given ID
func (c *Collection) Put(id string, content interface{}) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.options.TransactionTimeOut)
	defer cancel()

	// verify that closing as not been called
	if !c.isRunning() {
		return ErrClosedDB
	}

	tr := newTransaction(ctx)
	trElem := newTransactionElement(id, content, true, c)

	tr.addTransaction(trElem)

	// Run the insertion
	c.writeTransactionChan <- tr

	// And wait for the end of the insertion
	return <-tr.responseChan
}

// PutMulti put the given elements in the DB with one single write transaction.
// This must have much better performances than with multiple *Collection.Put().
// The number of IDs and of content must be equal.
func (c *Collection) PutMulti(IDs []string, content []interface{}) error {
	// Check the length of the parameters
	if len(IDs) != len(content) {
		return ErrPutMultiWrongLen
	}

	ctx, cancel := context.WithTimeout(c.ctx, c.options.TransactionTimeOut)
	defer cancel()

	// verify that closing as not been called
	if !c.isRunning() {
		return ErrClosedDB
	}

	tr := newTransaction(ctx)
	tr.transactions = make([]*writeTransactionElement, len(IDs))

	for i := range IDs {
		tr.transactions[i] = newTransactionElement(
			IDs[i],
			content[i],
			true,
			c,
		)
	}

	// Run the insertion
	c.writeTransactionChan <- tr
	// And wait for the end of the insertion
	return <-tr.responseChan
}

// Get retrieves the content of the given ID
func (c *Collection) Get(id string, pointer interface{}) (contentAsBytes []byte, _ error) {
	if id == "" {
		return nil, ErrEmptyID
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.options.TransactionTimeOut)
	defer cancel()

	response, getErr := c.get(ctx, id)
	if getErr != nil {
		return nil, getErr
	}
	contentAsBytes = response[0]

	if len(contentAsBytes) == 0 {
		return nil, fmt.Errorf("content of %q is empty or not present", id)
	}

	if pointer == nil {
		return contentAsBytes, nil
	}

	decoder := json.NewDecoder(bytes.NewBuffer(contentAsBytes))
	decoder.UseNumber()

	uMarshalErr := decoder.Decode(pointer)
	if uMarshalErr != nil {
		return nil, uMarshalErr
	}

	return contentAsBytes, nil
}

// Delete removes the corresponding object if the given ID
func (c *Collection) Delete(id string) error {
	ctx, cancel := context.WithTimeout(c.ctx, c.options.TransactionTimeOut)
	defer cancel()

	// verify that closing as not been called
	if !c.isRunning() {
		return ErrClosedDB
	}

	tr := newTransaction(ctx)
	trElem := newTransactionElement(id, nil, false, c)

	tr.addTransaction(trElem)

	// Run the insertion
	c.writeTransactionChan <- tr
	// And wait for the end of the insertion
	return <-tr.responseChan
}

// GetIDs returns a list of IDs for the given collection and starting
// at the given ID. The limit paramiter let caller ask for a portion of the collection.
func (c *Collection) GetIDs(startID string, limit int) ([]string, error) {
	records, getElemErr := c.getStoredIDsAndValues(startID, limit, true)
	if getElemErr != nil {
		return nil, getElemErr
	}

	ret := make([]string, len(records))
	for i, record := range records {
		ret[i] = record.GetID()
	}
	return ret, nil
}

// GetValues returns a list of IDs and values as bytes for the given collection and starting
// at the given ID. The limit paramiter let caller ask for a portion of the collection.
func (c *Collection) GetValues(startID string, limit int) ([]*ResponseElem, error) {
	return c.getStoredIDsAndValues(startID, limit, false)
}

// Rollback reset content to a previous version for the given key.
// The database by default keeps 10 version of the same key.
// previousVersion provide a way to get the wanted version where 0 is the fist previous
// content and bigger previousVersion is older the content will be.
// It returns the previous asked version timestamp.
// Everytime this function is called a new version is added.
func (c *Collection) Rollback(id string, previousVersion uint) (timestamp uint64, err error) {
	var contentAsInterface interface{}

	err = c.store.View(func(txn *badger.Txn) error {
		// Init the iterator
		iterator := txn.NewIterator(
			badger.IteratorOptions{
				AllVersions:    true,
				PrefetchSize:   c.options.BadgerOptions.NumVersionsToKeep,
				PrefetchValues: true,
			},
		)
		defer iterator.Close()

		// Set the rollback to at least the immediate previous content
		previousVersion = previousVersion + 1

		// Seek to the wanted key
		// Loop to the version
		for iterator.Seek(c.buildStoreID(id)); iterator.Valid(); iterator.Next() {
			item := iterator.Item()

			if !reflect.DeepEqual(c.buildStoreID(id), item.Key()) {
				return ErrRollbackVersionNotFound
			} else if previousVersion == 0 {
				item := item

				var asEncryptedBytes []byte
				asEncryptedBytes, err = item.ValueCopy(asEncryptedBytes)
				if err != nil {
					return err
				}
				var asBytes []byte
				asBytes, err = cipher.Decrypt(c.options.privateCryptoKey, item.Key(), asEncryptedBytes)
				if err != nil {
					return err
				}

				// Build a custom decoder to use the number interface instead of float64
				decoder := json.NewDecoder(bytes.NewBuffer(asBytes))
				decoder.UseNumber()

				decoder.Decode(&contentAsInterface)

				timestamp = item.Version()
				return nil
			}
			previousVersion--
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return timestamp, c.Put(id, contentAsInterface)
}

func (c *Collection) SetIndex(name string, bleveMapping mapping.IndexMapping) error {
	for _, i := range c.indexes {
		if i.Name == name {
			return ErrIndexNameAllreadyExists
		}
	}

	i := new(index)
	i.Name = name

	// Set the prefix
	i.Prefix = c.freePrefix[0]

	// Remove the prefix from the list of free prefixes
	c.freePrefix = append(c.freePrefix[:0], c.freePrefix[1:]...)

	kvConfig := c.buildKvConfig(i.Prefix)
	bleveIndex, err := bleve.NewUsing(name, bleveMapping, upsidedown.Name, blevestore.Name, kvConfig)
	if err != nil {
		return err
	}

	bleveIndex.Close()

	c.indexes = append(c.indexes, i)

	return c.saveCollections()
}

func (c *Collection) GetIndex(name string) (bleve.Index, error) {
	index, err := c.getIndex(name)
	if err != nil {
		return nil, err
	}

	return index.index, nil
}

func (c *Collection) DeleteIndex(name string) error {
	var index *index
	// Loop all indexes to found the given index
	found := false
	for j, i := range c.indexes {
		if i.Name == name {
			index = i
			found = true

			// Clean the slice of indexes
			copy(c.indexes[j:], c.indexes[j+1:])
			c.indexes[len(c.indexes)-1] = nil // or the zero value of T
			c.indexes = c.indexes[:len(c.indexes)-1]
			break
		}
	}

	if !found {
		return ErrIndexNotFound
	}

	return c.store.Update(func(txn *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.PrefetchValues = false
		iter := txn.NewIterator(opt)

		for iter.Seek(index.buildPrefix()); iter.ValidForPrefix(index.buildPrefix()); iter.Next() {
			item := iter.Item()

			var key []byte
			key = item.KeyCopy(key)
			err := txn.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

