package storage

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

var systemBucketName = []byte("system")
var binlogFileKey = []byte("binlogfile")
var binlogPositionKey = []byte("binlogpos")

// BoltDBStore Represents an instance of the BoltDB storage.
type BoltDBStore struct {
	db           *bolt.DB
	systemBucket *bolt.Bucket
}

// BinlogInformation Represents the information about the MySQL binlog.
type BinlogInformation struct {
	File     string
	Position uint32
}

// Open Opens the disk storage.
func (store *BoltDBStore) Open(filename string) error {
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.
	var err error
	store.db, err = bolt.Open(filename, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}

	// Ensure system bucket is created.
	store.db.Update(func(tx *bolt.Tx) error {
		store.systemBucket, err = tx.CreateBucketIfNotExists(systemBucketName)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return nil
}

// Close Closes the BoltDB storage.
func (store *BoltDBStore) Close() {
	store.db.Close()
}

// GetBinlogPosition Returns the persisted binlog position.
func (store *BoltDBStore) GetBinlogPosition() (*BinlogInformation, error) {
	binlogInfo := &BinlogInformation{}

	err := store.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(systemBucketName)
		file := bucket.Get(binlogFileKey)
		position := bucket.Get(binlogPositionKey)

		if file == nil || position == nil {
			return errors.New("binlog file and position not found")
		}
		binlogInfo.File = string(file)
		binlogInfo.Position = binary.LittleEndian.Uint32(position)
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "unable to open readonly transaction")
	}
	return binlogInfo, nil
}

// SetBinlogPosition Sets and persists the current binlog position.
func (store *BoltDBStore) SetBinlogPosition(binlogInfo *BinlogInformation) error {
	fileBuffer := []byte(binlogInfo.File)
	positionBuffer := make([]byte, 4)
	var err error

	binary.LittleEndian.PutUint32(positionBuffer, binlogInfo.Position)

	store.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(systemBucketName)
		err = bucket.Put([]byte(binlogFileKey), fileBuffer)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(binlogPositionKey), positionBuffer)
		if err != nil {
			return err
		}
		return nil
	})

	err = store.db.Sync()
	if err != nil {
		return err
	}
	return nil
}