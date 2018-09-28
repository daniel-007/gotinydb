package gotinydb

import (
	"context"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
)

type (
	// DB is the main element of the package and provide all access to sub commands
	DB struct {
		options *Options

		badgerDB    *badger.DB
		collections []*Collection

		// freePrefix defines the list of prefix which can be used for a new collections
		freePrefix []byte

		writeTransactionChan chan *writeTransaction

		ctx     context.Context
		closing bool
	}

	dbExport struct {
		Collections      []*collectionExport
		FreePrefix       []byte
		PrivateCryptoKey [32]byte
	}
	collectionExport struct {
		Name    string
		Prefix  byte
		Indexes []*index
	}

	// Options defines the deferent configuration elements of the database
	Options struct {
		Path                             string
		TransactionTimeOut, QueryTimeOut time.Duration
		InternalQueryLimit               int
		// This define the limit which apply to the serialization of the writes
		PutBufferLimit int

		// CryptoKey if present must be 32 bytes long, Otherwise an empty key is used.
		CryptoKey [32]byte
		// privateCryptoKey is saved on the database to provide a way to change the password
		// without the need to rewrite the all database
		privateCryptoKey [32]byte

		// GCCycle define the time the loop for garbage collection takes to run the GC.
		GCCycle time.Duration

		FileChunkSize int

		BadgerOptions *badger.Options
	}

	// Collection defines the storage object
	Collection struct {
		name string

		// prefix defines the prefix needed to found the collection into the store
		prefix byte

		// freePrefix defines the list of prefix which can be used for a new indexes
		freePrefix []byte

		indexes []*index

		options *Options

		store *badger.DB

		writeTransactionChan chan *writeTransaction

		ctx context.Context

		saveCollections func() error
	}

	index struct {
		Name   string
		Prefix byte
		Path   string

		kvConfig map[string]interface{}

		collectionPrefix byte

		index bleve.Index
	}

	// SearchResult is returned when (*Collection).Shearch is call.
	// It contains the result and a iterator for the reading values directly from database.
	SearchResult struct {
		BleveSearchResult *bleve.SearchResult

		position uint64
		c        *Collection

		// preload      uint
		// preloaded    [][]byte
		// preloadedErr []error
	}

	// idType is a type to order IDs during query to be compatible with the tree query
	idType struct {
		ID          string
		occurrences int
		ch          chan int
	}

	// idsType defines a list of ID. The struct is needed to build a pointer to be
	// passed to deferent functions
	idsType struct {
		IDs []*idType
	}

	idsTypeMultiSorter struct {
		IDs    []*idType
		invert bool
	}

	// Response holds the results of a query
	Response struct {
		list           []*ResponseElem
		actualPosition int
	}

	// ResponseElem defines the response as a pointer
	ResponseElem struct {
		_ID            *idType
		contentAsBytes []byte
	}

	writeTransaction struct {
		responseChan chan error
		ctx          context.Context
		transactions []*writeTransactionElement
	}
	writeTransactionElement struct {
		id               string
		collection       *Collection
		contentInterface interface{}
		contentAsBytes   []byte
		chunkN           int
		isInsertion      bool
		isFile           bool
	}
)
