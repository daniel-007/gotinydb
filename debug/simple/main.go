package simple

import "fmt"

// Those constants defines the first level of prefixes.
const (
	prefixConfig byte = iota
	prefixCollections
	prefixFiles
)

// Those constants defines the second level of prefixes or value from config.
const (
	prefixCollectionsData byte = iota
	prefixCollectionsBleveIndex
)

var (
	ErrNotFound = fmt.Errorf("value not found")
)