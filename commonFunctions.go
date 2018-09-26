package gotinydb

import (
	"context"
	"encoding/json"
)

func getIDsAsString(input []*idType) (ret []string) {
	for _, id := range input {
		ret = append(ret, id.ID)
	}
	return ret
}

func newTransactionElement(id string, content interface{}, isInsertion bool, col *Collection) (wtElem *writeTransactionElement) {
	wtElem = &writeTransactionElement{
		id: id, contentInterface: content, isInsertion: isInsertion, collection: col,
	}

	if !isInsertion {
		return
	}

	jsonBytes, marshalErr := json.Marshal(content)
	if marshalErr != nil {
		return nil
	}

	wtElem.contentAsBytes = jsonBytes

	return
}

func newFileTransactionElement(id string, chunkN int, content []byte, isInsertion bool) *writeTransactionElement {
	return &writeTransactionElement{
		id: id, chunkN: chunkN, contentAsBytes: content, isInsertion: isInsertion, isFile: true,
	}
}

func newTransaction(ctx context.Context) *writeTransaction {
	wt := new(writeTransaction)
	wt.ctx = ctx
	wt.responseChan = make(chan error, 0)

	return wt
}

func (wt *writeTransaction) addTransaction(trElement ...*writeTransactionElement) {
	wt.transactions = append(wt.transactions, trElement...)
}

// // buildSelectorHash returns a string hash of the selector
// func buildSelectorHash(selector []string) uint16 {
// 	hasher, _ := blake2b.New256(nil)
// 	for _, filedName := range selector {
// 		hasher.Write([]byte(filedName))
// 	}

// 	hash := binary.BigEndian.Uint16(hasher.Sum(nil))
// 	return hash
// }
