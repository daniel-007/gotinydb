package gotinydb

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
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

func (c *Collection) Search(indexName string, searchRequest *bleve.SearchRequest) (*SearchResult, error) {
	ret := new(SearchResult)

	bleveIndex, err := c.GetIndex(indexName)
	if err != nil {
		return nil, err
	}

	ret.BleveSearchResult, err = bleveIndex.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	ret.c = c

	return ret, nil
}

func (s *SearchResult) Next(dest interface{}) (*search.DocumentMatch, error) {
	if s.BleveSearchResult.Total-1 < s.position {
		return nil, ErrSearchOver
	}

	doc := s.BleveSearchResult.Hits[s.position]

	_, err := s.c.Get(doc.ID, dest)
	if err != nil {
		return nil, err
	}

	s.position++

	return doc, nil
}
