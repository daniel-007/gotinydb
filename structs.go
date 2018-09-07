package gotinydb

import (
	"context"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/dgraph-io/badger"
	"github.com/lucas-clemente/quic-go/h2quic"
)

type (
	// Replica defines the element to interact with the rest of the cluster
	Replica struct {
		Master  bool
		Address string
	}

	// DB is the main element of the package and provide all access to sub commands
	DB struct {
		options *Options

		valueStore  *badger.DB
		collections []*Collection

		master          *Replica
		slaves          []*Replica
		netWorkListener *h2quic.Server

		ctx     context.Context
		closing bool
	}

	// Options defines the deferent configuration elements of the database
	Options struct {
		Path                             string
		TransactionTimeOut, QueryTimeOut time.Duration
		InternalQueryLimit               int
		// This define the limit which apply to the serialization of the writes
		PutBufferLimit int

		AddressBindNetworkService string

		BadgerOptions *badger.Options
		BoltOptions   *bolt.Options
	}

	// Collection defines the storage object
	Collection struct {
		name, id string
		indexes  []*indexType

		options *Options

		db    *bolt.DB
		store *badger.DB

		writeTransactionChan chan *writeTransaction

		ctx context.Context
	}

	// Filter defines the way the query will be performed
	Filter struct {
		selector     []string
		selectorHash uint64
		operator     FilterOperator
		values       []*filterValue
		equal        bool
		exclusion    bool
	}

	// IndexType defines what kind of field the index is scanning
	IndexType int

	// filterValue defines the value we need to compare to
	filterValue struct {
		Value interface{}
		Type  IndexType
	}

	// Index defines the struct to manage indexation
	indexType struct {
		Name         string
		Selector     []string
		SelectorHash uint64
		Type         IndexType

		options *Options

		getTx func(update bool) (*bolt.Tx, error)
	}

	// refs defines an struct to manage the references of a given object
	// in all the indexes it belongs to
	refs struct {
		ObjectID     string
		ObjectHashID string

		Refs []*ref
	}

	// ref defines the relations between a object with some index with indexed value
	ref struct {
		IndexName    string
		IndexHash    uint64
		IndexedValue []byte
	}

	writeTransaction struct {
		responseChan chan error
		ctx          context.Context
		transactions []*writeTransactionElement
	}
	writeTransactionElement struct {
		id               string
		contentInterface interface{}
		contentAsBytes   []byte
		bin              bool
	}

	// Archive defines the way archives are saved inside the zip file
	archive struct {
		StartTime, EndTime time.Time
		Indexes            map[string][]*indexType
		Collections        []string
		Timestamp          uint64

		file *os.File
	}

	// IndexInfo is returned by *Collection.GetIndexesInfo and let call see
	// what indexes are present in the collection.
	IndexInfo struct {
		Name     string
		Selector []string
		Type     IndexType
	}
)
