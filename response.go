package gotinydb

import (
	"bytes"
	"encoding/json"
)

// newResponse build a new Response pointer with the given limit
func newResponse(limit int) *Response {
	r := new(Response)
	r.list = make([]*ResponseElem, limit)
	return r
}

// Len returns the length of the given response
func (r *Response) Len() int {
	return len(r.list)
}

// First used with Next
func (r *Response) First() (i int, id string, objAsByte []byte) {
	r.actualPosition = 0
	return 0, r.list[0].GetID(), r.list[0].contentAsBytes
}

// Next used with First
func (r *Response) Next() (i int, id string, objAsByte []byte) {
	r.actualPosition++
	return r.next()
}

// Last used with Prev
func (r *Response) Last() (i int, id string, objAsByte []byte) {
	lastSlot := len(r.list) - 1

	r.actualPosition = lastSlot
	return lastSlot, r.list[lastSlot].GetID(), r.list[lastSlot].contentAsBytes
}

// Prev used with Last
func (r *Response) Prev() (i int, id string, objAsByte []byte) {
	r.actualPosition--
	return r.next()
}

// Is called by r.Next r.Prev to get their next values
func (r *Response) next() (i int, id string, objAsByte []byte) {
	if r.actualPosition >= len(r.list) || r.actualPosition < 0 {
		r.actualPosition = 0
		return -1, "", nil
	}
	return r.actualPosition, r.list[r.actualPosition].GetID(), r.list[r.actualPosition].contentAsBytes
}

// All takes a function as argument and permit to unmarshal or to manage recoredes inside the function
func (r *Response) All(fn func(id string, objAsBytes []byte) error) (n int, err error) {
	for _, elem := range r.list {
		err = fn(elem.GetID(), elem.contentAsBytes)
		if err != nil {
			return
		}
	}
	return
}

// One retrieve one element at the time and put it into the destination pointer.
// Use it to get the objects one after the other.
func (r *Response) One(destination interface{}) (id string, err error) {
	if r.actualPosition >= len(r.list) {
		r.actualPosition = 0
		return "", ErrResponseOver
	}

	id = r.list[r.actualPosition].GetID()

	decoder := json.NewDecoder(bytes.NewBuffer(r.list[r.actualPosition].contentAsBytes))
	decoder.UseNumber()

	err = decoder.Decode(destination)
	r.actualPosition++

	return id, err
}

// GetID returns the ID as string of the given element
func (r *ResponseElem) GetID() string {
	return r._ID.ID
}

// GetContent returns response content as a slice of bytes
func (r *ResponseElem) GetContent() []byte {
	return r.contentAsBytes
}

// Unmarshal tries to unmarshal the content using the JSON package
func (r *ResponseElem) Unmarshal(pointer interface{}) (err error) {
	decoder := json.NewDecoder(bytes.NewBuffer(r.contentAsBytes))
	decoder.UseNumber()

	return decoder.Decode(pointer)
}