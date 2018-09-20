package gotinydb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/dgraph-io/badger"
)

// Put add the given content to database with the given ID
func (c *Collection) Put(id string, content interface{}) error {
	return c.put(id, newTransactionElement(id, content, true, c))
}

// PutTTL does same as Put but with an expiration to the record
func (c *Collection) PutTTL(id string, content interface{}, duration time.Duration) error {
	trElem := newTransactionElement(id, content, true, c)
	trElem.ttl = time.Now().Add(duration)
	return c.put(id, trElem)
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

// SetIndex enable the collection to index field or sub field
func (c *Collection) SetIndex(name string, t IndexType, selector ...string) error {
	for _, index := range c.indexes {
		if index.Name == name {
			return ErrIndexNameAllreadyExists
		}
	}

	i := newIndex(name, t, selector...)
	i.options = c.options
	i.getTx = c.store.NewTransaction
	i.getIDBuilder = func(id []byte) []byte {
		return c.buildIDWhitPrefixIndex([]byte(i.Name), id)
	}

	c.indexes = append(c.indexes, i)

	err := c.indexAllValues()
	if err != nil {
		return err
	}

	return c.saveCollections()
}

// DeleteIndex remove the index from the collection
func (c *Collection) DeleteIndex(name string) error {
	txn := c.store.NewTransaction(true)
	defer txn.Discard()
	// Find the correct index from the list
	for i, activeIndex := range c.indexes {
		if activeIndex.Name == name {
			// Clean the collection list from the index pointer
			copy(c.indexes[i:], c.indexes[i+1:])
			c.indexes[len(c.indexes)-1] = nil
			c.indexes = c.indexes[:len(c.indexes)-1]

			// Remove all indexed values from database
			iteratorOptions := badger.DefaultIteratorOptions
			iteratorOptions.PrefetchValues = false
			iterator := txn.NewIterator(iteratorOptions)
			// Make sure that the iterator is closed.
			// But we have to make sure that close is called only onces
			// but we need to run it before commit.
			defer func() {
				if r := recover(); r != nil {
					iterator.Close()
				}
			}()

			indexPrefix := c.buildIDWhitPrefixIndex([]byte(name), nil)
			for iterator.Seek(indexPrefix); iterator.ValidForPrefix(indexPrefix); iterator.Next() {
				err := txn.Delete(iterator.Item().Key())
				if err != nil {
					return err
				}
			}

			// Need to close the iterator before commit
			iterator.Close()

			err := txn.Commit(nil)
			if err != nil {
				return err
			}

			return c.saveCollections()
		}
	}

	return ErrNotFound
}

// Query run the given query to all the collection indexes
func (c *Collection) Query(q *Query) (response *Response, _ error) {
	if q == nil {
		return
	}

	// If no filter the query stops
	if len(q.filters) <= 0 {
		return nil, fmt.Errorf("query has no filter")
	}

	// If no index stop the query
	if len(c.indexes) <= 0 {
		return nil, fmt.Errorf("no index in the collection")
	}

	if q.internalLimit > c.options.InternalQueryLimit {
		q.internalLimit = c.options.InternalQueryLimit
	}
	if q.timeout > c.options.QueryTimeOut {
		q.timeout = c.options.QueryTimeOut
	}

	// Set a timout
	ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	tree, err := c.queryGetIDs(ctx, q)
	if err != nil {
		return nil, err
	}

	return c.queryCleanAndOrder(ctx, q, tree)
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
			if !reflect.DeepEqual(c.buildStoreID(id), iterator.Item().Key()) {
				return ErrRollbackVersionNotFound
			} else if previousVersion == 0 {
				item := iterator.Item()

				var asEncryptedBytes []byte
				asEncryptedBytes, err = item.Value()
				if err != nil {
					return err
				}
				var asBytes []byte
				asBytes, err = decrypt(c.options.privateCryptoKey, item.Key(), asEncryptedBytes)
				if err != nil {
					return err
				}

				// Build a custom decoder to use the number interface instead of float64
				decoder := json.NewDecoder(bytes.NewBuffer(asBytes))
				decoder.UseNumber()

				unmarshalErr := decoder.Decode(&contentAsInterface)
				if unmarshalErr != nil {
					return unmarshalErr
				}

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

	putErr := c.Put(id, contentAsInterface)
	if putErr != nil {
		return 0, putErr
	}

	return timestamp, nil
}

// GetIndexesInfo retruns a slice with indexes settings
func (c *Collection) GetIndexesInfo() []*IndexInfo {
	indexesInfo := make([]*IndexInfo, len(c.indexes))
	for i := 0; i < len(c.indexes); i++ {
		indexInfo := &IndexInfo{
			Name:     c.indexes[i].Name,
			Selector: c.indexes[i].Selector,
			Type:     c.indexes[i].Type,
		}
		indexesInfo[i] = indexInfo
	}

	return indexesInfo
}
