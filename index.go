package gotinydb

import (
	"github.com/blevesearch/bleve"
)

func (i *index) open() error {
	if i.index != nil {
		return nil
	}

	bleveIndex, err := bleve.OpenUsing(i.Path, i.kvConfig)
	if err != nil {
		return err
	}

	i.index = bleveIndex

	return nil
}

func (i *index) buildPrefix() []byte {
	return []byte{i.collectionPrefix, i.Prefix}
}
