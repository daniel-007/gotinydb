package gotinydb

func (i *index) buildPrefix() []byte {
	return []byte{i.collectionPrefix, i.Prefix}
}
