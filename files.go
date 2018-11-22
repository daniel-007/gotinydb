package gotinydb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/alexandrestein/gotinydb/transaction"
	"github.com/dgraph-io/badger"
	"golang.org/x/crypto/blake2b"
)

type (
	// FileMeta defines some file metadata informations
	FileMeta struct {
		ID           string
		Name         string
		Size         int64
		LastModified time.Time
		ChuckSize    int
		inWrite      bool
	}

	readWriter struct {
		meta            *FileMeta
		db              *DB
		currentPosition int64
		txn             *badger.Txn
		writer          bool
	}

	// Reader define a simple object to read parts of the file
	Reader interface {
		io.ReadCloser
		io.Seeker
		io.ReaderAt

		GetMeta() *FileMeta
	}

	// Writer define a simple object to write parts of the file
	Writer interface {
		Reader

		io.Writer
		io.WriterAt
	}
)

// PutFile let caller insert large element into the database via a reader interface
func (d *DB) PutFile(id string, name string, reader io.Reader) (n int, err error) {
	d.DeleteFile(id)

	meta := d.buildMeta(id, name)
	meta.inWrite = true

	// Set the meta
	err = d.putFileMeta(meta)
	if err != nil {
		return
	}

	// Track the numbers of chunks
	nChunk := 1
	// Open a loop
	for true {
		// Initialize the read buffer
		buff := make([]byte, FileChuckSize)
		var nWritten int
		nWritten, err = reader.Read(buff)
		// The read is done and it returns
		if nWritten == 0 || err == io.EOF && nWritten == 0 {
			break
		}
		// Return error if any
		if err != nil && err != io.EOF {
			return
		}

		// Clean the buffer
		buff = buff[:nWritten]

		n = n + nWritten

		err = d.writeFileChunk(id, nChunk, buff)
		if err != nil {
			return n, err
		}

		// Increment the chunk counter
		nChunk++
	}

	meta.Size = int64(n)
	meta.LastModified = time.Now()
	meta.inWrite = false
	err = d.putFileMeta(meta)
	if err != nil {
		return
	}

	err = nil
	return
}

func (d *DB) writeFileChunk(id string, chunk int, content []byte) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if FileChuckSize < len(content) {
		return fmt.Errorf("the maximum chunk size is %d bytes long but the content to write is %d bytes long", FileChuckSize, len(content))
	}

	tx := transaction.New(ctx)
	tx.AddOperation(
		transaction.NewOperation("", nil, d.buildFilePrefix(id, chunk), content, false, true),
	)
	// Run the insertion
	select {
	case d.writeChan <- tx:
	case <-d.ctx.Done():
		return d.ctx.Err()
	}

	// And wait for the end of the insertion
	select {
	case err = <-tx.ResponseChan:
	case <-tx.Ctx.Done():
		err = tx.Ctx.Err()
	}
	return
}

func (d *DB) getFileMeta(id, name string) (meta *FileMeta, err error) {
	meta = new(FileMeta)
	var caller *GetCaller
	caller, err = d.buildGetCaller(d.buildFilePrefix(id, 0), meta)
	if err != nil {
		return
	}

	err = d.get(caller)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			err = nil
			meta = d.buildMeta(id, name)
			return
		}
		return nil, err
	}

	return meta, nil
}

func (d *DB) buildMeta(id, name string) (meta *FileMeta) {
	meta = new(FileMeta)
	meta.ID = id
	meta.Name = name
	meta.Size = 0
	meta.LastModified = time.Now()
	meta.ChuckSize = FileChuckSize

	return
}

func (d *DB) putFileMeta(meta *FileMeta) (err error) {
	metaID := d.buildFilePrefix(meta.ID, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var metaAsBytes []byte
	metaAsBytes, err = json.Marshal(meta)
	if err != nil {
		return
	}

	tx := transaction.New(ctx)
	tx.AddOperation(
		transaction.NewOperation("", nil, metaID, metaAsBytes, false, false),
	)
	// Run the insertion
	select {
	case d.writeChan <- tx:
	case <-d.ctx.Done():
		return d.ctx.Err()
	}
	// And wait for the end of the insertion
	select {
	case err = <-tx.ResponseChan:
	case <-tx.Ctx.Done():
		err = tx.Ctx.Err()
	}
	return
}

// ReadFile write file content into the given writer
func (d *DB) ReadFile(id string, writer io.Writer) error {
	return d.badger.View(func(txn *badger.Txn) error {
		storeID := d.buildFilePrefix(id, -1)

		opt := badger.DefaultIteratorOptions
		opt.PrefetchSize = 3
		opt.PrefetchValues = true

		it := txn.NewIterator(opt)
		defer it.Close()

		for it.Seek(d.buildFilePrefix(id, 1)); it.ValidForPrefix(storeID); it.Next() {
			var err error
			var valAsEncryptedBytes []byte
			valAsEncryptedBytes, err = it.Item().ValueCopy(valAsEncryptedBytes)
			if err != nil {
				return err
			}

			var valAsBytes []byte
			valAsBytes, err = d.decryptData(it.Item().Key(), valAsEncryptedBytes)
			if err != nil {
				return err
			}

			_, err = writer.Write(valAsBytes)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// GetFileReader returns a struct to provide simple reading partial of big files.
// The default position is at the begining of the file.
func (d *DB) GetFileReader(id string) (Reader, error) {
	rw, err := d.newReadWriter(id, "", false)
	return Reader(rw), err
}

// GetFileWriter returns a struct to provide simple partial write of big files.
// The default position is at the end of the file.
func (d *DB) GetFileWriter(id, name string) (Writer, error) {
	rw, err := d.newReadWriter(id, name, true)
	if err != nil {
		return nil, err
	}

	if rw.meta.inWrite {
		return nil, ErrFileInWrite
	}

	rw.meta.inWrite = true
	err = d.putFileMeta(rw.meta)
	if err != nil {
		return nil, err
	}

	rw.currentPosition = rw.meta.Size
	return Writer(rw), err
}

// DeleteFile deletes every chunks of the given file ID
func (d *DB) DeleteFile(id string) (err error) {
	listOfTx := []*transaction.Transaction{}

	// Open a read transaction to get every IDs
	return d.badger.View(func(txn *badger.Txn) error {
		// Build the file prefix
		storeID := d.buildFilePrefix(id, -1)

		// Defines the iterator options to get only IDs
		opt := badger.DefaultIteratorOptions
		opt.PrefetchValues = false

		// Initialize the iterator
		it := txn.NewIterator(opt)
		defer it.Close()

		// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Go the the first file chunk
		for it.Seek(storeID); it.ValidForPrefix(storeID); it.Next() {
			// Copy the store key
			var key []byte
			key = it.Item().KeyCopy(key)
			// And add it to the list of store IDs to delete
			tx := transaction.New(ctx)
			tx.AddOperation(
				transaction.NewOperation("", nil, key, nil, true, true),
			)
			listOfTx = append(listOfTx, tx)
			d.writeChan <- tx
		}

		for _, tx := range listOfTx {
			select {
			case err = <-tx.ResponseChan:
			case <-tx.Ctx.Done():
				err = tx.Ctx.Err()
			}
			if err != nil {
				return err
			}
		}

		// Close the view transaction
		return nil
	})
}

func (d *DB) buildFilePrefix(id string, chunkN int) []byte {
	// Derive the ID to make sure no file ID overlap the other.
	// Because the files are chunked it needs to have a stable prefix for reading
	// and deletation.
	derivedID := blake2b.Sum256([]byte(id))

	// Build the prefix
	prefixWithID := append([]byte{prefixFiles}, derivedID[:]...)

	// Initialize the chunk part of the ID
	chunkPart := []byte{}

	// If less than zero it for deletation and only the prefix is returned
	if chunkN < 0 {
		return prefixWithID
	}

	// If it's the first chunk
	if chunkN == 0 {
		chunkPart = append(chunkPart, 0)
	} else {
		// Lockup the numbers of full bytes for the chunk ID
		nbFull := chunkN / 256
		restFull := chunkN % 256

		for index := 0; index < nbFull; index++ {
			chunkPart = append(chunkPart, 255)
		}
		chunkPart = append(chunkPart, uint8(restFull))
	}

	// Return the ID for the given file and ID
	return append(prefixWithID, chunkPart...)
}

func (d *DB) newReadWriter(id, name string, writer bool) (_ *readWriter, err error) {
	rw := new(readWriter)
	rw.writer = writer

	rw.meta, err = d.getFileMeta(id, name)
	if err != nil {
		return nil, err
	}

	rw.db = d
	rw.txn = d.badger.NewTransaction(false)

	return rw, nil
}

// Read implements the io.Reader interface
func (r *readWriter) Read(p []byte) (n int, err error) {
	block, inside := r.getBlockAndInsidePosition(r.currentPosition)

	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = 3
	opt.PrefetchValues = true

	it := r.txn.NewIterator(opt)
	defer it.Close()

	buffer := bytes.NewBuffer(nil)
	first := true

	filePrefix := r.db.buildFilePrefix(r.meta.ID, -1)
	for it.Seek(r.db.buildFilePrefix(r.meta.ID, block)); it.ValidForPrefix(filePrefix); it.Next() {
		if it.Item().IsDeletedOrExpired() {
			break
		}

		var err error
		var valAsEncryptedBytes []byte
		valAsEncryptedBytes, err = it.Item().ValueCopy(valAsEncryptedBytes)
		if err != nil {
			return 0, err
		}

		var valAsBytes []byte
		valAsBytes, err = r.db.decryptData(it.Item().Key(), valAsEncryptedBytes)
		if err != nil {
			return 0, err
		}

		var toAdd []byte
		if first {
			toAdd = valAsBytes[inside:]
		} else {
			toAdd = valAsBytes
		}
		buffer.Write(toAdd)
		if buffer.Len() >= len(p) {
			copy(p, buffer.Bytes()[:len(p)])
			r.currentPosition += int64(len(p))
			return len(p), nil
		}

		first = false
	}

	copy(p, buffer.Bytes())

	r.currentPosition = 0

	return buffer.Len(), io.EOF
}

func (r *readWriter) checkReadWriteAt(off int64) error {
	if r.meta.Size <= off {
		return fmt.Errorf("the offset can not be equal or bigger than the file")
	}
	return nil
}

// ReadAt implements the io.ReaderAt interface
func (r *readWriter) ReadAt(p []byte, off int64) (n int, err error) {
	err = r.checkReadWriteAt(off)
	if err != nil {
		return 0, err
	}

	r.currentPosition = off
	return r.Read(p)
}

func (r *readWriter) getExistingBlock(blockN int) ([]byte, error) {
	chunkID := r.db.buildFilePrefix(r.meta.ID, blockN)

	caller, err := r.db.Get(chunkID)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return []byte{}, nil
		}
		return nil, err
	}

	return caller.asBytes, nil
}

func (r *readWriter) Write(p []byte) (n int, err error) {
	// Get a new transaction to be able to call write multiple times
	defer r.afterWrite(len(p))

	block, inside := r.getBlockAndInsidePosition(r.currentPosition)

	var valAsBytes []byte
	valAsBytes, err = r.getExistingBlock(block)
	if err != nil {
		return 0, err
	}

	freeToWriteInThisChunk := FileChuckSize - inside
	if freeToWriteInThisChunk > len(p) {
		toWrite := []byte{}
		if inside <= len(valAsBytes) {
			toWrite = valAsBytes[:inside]
		}
		toWrite = append(toWrite, p...)

		// If the new content don't completely overwrite the previous content
		if existingAfterNewWriteStartPosition := inside + len(p); existingAfterNewWriteStartPosition < len(valAsBytes) {
			toWrite = append(toWrite, valAsBytes[existingAfterNewWriteStartPosition:]...)
		}

		return len(p), r.db.writeFileChunk(r.meta.ID, block, toWrite)
	}

	toWriteInTheFirstChunk := valAsBytes[:inside]
	toWriteInTheFirstChunk = append(toWriteInTheFirstChunk, p[n:freeToWriteInThisChunk]...)
	err = r.db.writeFileChunk(r.meta.ID, block, toWriteInTheFirstChunk)
	if err != nil {
		return n, err
	}

	n += freeToWriteInThisChunk
	block++

	done := false

newLoop:
	newEnd := n + FileChuckSize
	if newEnd > len(p) {
		newEnd = len(p)
		done = true
	}

	nextToWrite := p[n:newEnd]
	if done {
		valAsBytes, err = r.getExistingBlock(block)
		if err != nil {
			return 0, err
		}
		nextToWrite = append(nextToWrite, valAsBytes[len(nextToWrite):]...)
	}

	err = r.db.writeFileChunk(r.meta.ID, block, nextToWrite)
	if err != nil {
		return n, err
	}

	n += FileChuckSize
	block++

	if done {
		n = len(p)
		return
	}

	goto newLoop
}

func (r *readWriter) afterWrite(writenLength int) {
	// Refrech the transaction
	r.txn.Discard()
	r.txn = r.db.badger.NewTransaction(false)

	r.meta.Size += r.getWrittenSize()
	r.meta.LastModified = time.Now()

	r.currentPosition += int64(writenLength)

	r.db.putFileMeta(r.meta)
}

func (r *readWriter) getWrittenSize() (n int64) {
	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = 5
	opt.PrefetchValues = false

	it := r.txn.NewIterator(opt)
	defer it.Close()

	nbChunks := -1
	blockesPrefix := r.db.buildFilePrefix(r.meta.ID, -1)
	var item *badger.Item

	var lastBlockItem *badger.Item
	for it.Seek(r.db.buildFilePrefix(r.meta.ID, 1)); it.ValidForPrefix(blockesPrefix); it.Next() {
		item = it.Item()
		if item.IsDeletedOrExpired() {
			break
		}
		lastBlockItem = item
		nbChunks++
	}

	if lastBlockItem == nil {
		return 0
	}

	var encryptedValue []byte
	var err error
	encryptedValue, err = lastBlockItem.ValueCopy(encryptedValue)
	if err != nil {
		return
	}

	var valAsBytes []byte
	valAsBytes, err = r.db.decryptData(item.Key(), encryptedValue)
	if err != nil {
		return
	}

	n = int64(nbChunks * r.meta.ChuckSize)
	n += int64(len(valAsBytes))

	return
}

func (r *readWriter) WriteAt(p []byte, off int64) (n int, err error) {
	err = r.checkReadWriteAt(off)
	if err != nil {
		return 0, err
	}

	r.currentPosition = off
	return r.Write(p)
}

// Seek implements the io.Seeker interface
func (r *readWriter) Seek(offset int64, whence int) (n int64, err error) {
	switch whence {
	case io.SeekStart:
		n = offset
	case io.SeekCurrent:
		n = r.currentPosition + offset
	case io.SeekEnd:
		n = r.meta.Size - offset
	default:
		err = fmt.Errorf("whence not recognized")
	}

	if n > r.meta.Size || n < 0 {
		err = fmt.Errorf("is out of the file")
	}

	r.currentPosition = n
	return
}

// Close should be called when done with the Reader
func (r *readWriter) Close() (err error) {
	if r.writer {
		r.meta.inWrite = false
		r.db.putFileMeta(r.meta)
	}
	r.txn.Discard()
	return
}

func (r *readWriter) GetMeta() *FileMeta {
	return r.meta
}

func (r *readWriter) getBlockAndInsidePosition(offset int64) (block, inside int) {
	return int(offset/int64(r.meta.ChuckSize)) + 1, int(offset) % r.meta.ChuckSize
}
