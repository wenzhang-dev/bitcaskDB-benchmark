package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/boltdb/bolt"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/stretchr/testify/assert"
	"github.com/wenzhang-dev/bitcaskDB"
)

func getBitcaskDB(b *testing.B) bitcask.DB {
	dir := "./bitcaskDB"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, os.ModePerm)

	opts := &bitcask.Options{
		Dir:                       dir,
		WalMaxSize:                1024 * 1024 * 1024, // 1GB
		ManifestMaxSize:           10 * 1024 * 1024,   // 10MB
		IndexCapacity:             10000000,           // 10 million
		IndexLimited:              8000000,
		IndexEvictionPoolCapacity: 64,
		IndexSampleKeys:           5,
		CompactionPicker:          nil, // default picker
		CompactionFilter:          nil, // default filter
		NsSize:                    0,
		EtagSize:                  0,
		DisableCompaction:         false,
	}

	db, err := bitcask.NewDB(opts)
	assert.Nil(b, err)
	return db
}

func getBadgerDB(b *testing.B) *badger.DB {
	dir := "./badger"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, os.ModePerm)

	opts := badger.DefaultOptions(dir).WithLoggingLevel(3)
	db, err := badger.Open(opts)
	assert.Nil(b, err)
	return db
}

func getLevelDB(b *testing.B) *leveldb.DB {
	dir := "./leveldb"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, os.ModePerm)

	ldb, err := leveldb.OpenFile(dir, nil)
	assert.Nil(b, err)
	return ldb
}

func getBoltDB(b *testing.B) *bolt.DB {
	dir := "./bolt"
	_ = os.RemoveAll(dir)
	_ = os.MkdirAll(dir, os.ModePerm)

	opts := bolt.DefaultOptions
	db, err := bolt.Open(filepath.Join(dir, "bolt"), 0o777, opts)
	// speed up writes
	db.NoSync = true
	assert.Nil(b, err)

	err = db.Update(func(txn *bolt.Tx) error {
		_, err := txn.CreateBucketIfNotExists([]byte("benchmark"))
		return err
	})
	assert.Nil(b, err)

	return db
}

func newKey(hint, threshold int) []byte {
	hint %= threshold
	key := fmt.Sprintf("key=%10d,%10d", hint, hint) // 25 bytes
	return []byte(key)
}

func randomKey(threshold int) []byte {
	k := rand.Int() % threshold
	key := fmt.Sprintf("key=%10d,%10d", k, k) // 25 bytes
	return []byte(key)
}

var (
	stopCh     chan struct{}
	rssSamples []uint64
)

func getRSS() (uint64, error) {
	data, err := ioutil.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, errors.New("invalid data")
	}

	rssPages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}

	return rssPages * uint64(os.Getpagesize()), nil
}

func sampleRSS(b *testing.B) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			rss, err := getRSS()
			assert.Nil(b, err)
			rssSamples = append(rssSamples, rss)
		}
	}
}

func getPeakRSS() uint64 {
	peak := uint64(0)
	for _, rss := range rssSamples {
		peak = max(peak, rss)
	}
	return peak
}

func getAvgRSS() uint64 {
	if len(rssSamples) == 0 {
		return 0
	}

	avg := uint64(0)
	for _, rss := range rssSamples {
		avg += rss
	}
	return avg / uint64(len(rssSamples))
}

func reportRss(b *testing.B) {
	avg := getAvgRSS()
	peak := getPeakRSS()

	b.ReportMetric(float64(avg)/1024/1024/1024, "AvgRSS(GB)")
	b.ReportMetric(float64(peak)/1024/1024/1024, "PeakRSS(GB)")
}

func BenchmarkReadWithBitcaskDB(b *testing.B) {
	db := getBitcaskDB(b)
	defer db.Close()

	// write 1 million
	threshold := 1000000
	meta := bitcask.NewMeta(nil)
	value4KB := bitcask.GenNKBytes(4)
	opts := &bitcask.WriteOptions{}

	batchSize := 50
	batch := bitcask.NewBatch()
	for i := 1; i <= threshold; i++ {
		batch.Put(nil, newKey(i, threshold), value4KB, meta)

		if i%batchSize == 0 {
			err := db.Write(batch, opts)
			assert.Nil(b, err)
			batch.Clear()
		}
	}

	b.Run("read4K", func(b *testing.B) {
		benchmarkReadWithBitcaskDB(b, db, threshold)
	})
}

func benchmarkReadWithBitcaskDB(b *testing.B, db bitcask.DB, threshold int) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		opts := &bitcask.ReadOptions{}
		for pb.Next() {
			_, _, err := db.Get(nil, randomKey(threshold), opts)
			assert.Nil(b, err)
		}
	})

	reportRss(b)
}

func BenchmarkReadWithBadger(b *testing.B) {
	db := getBadgerDB(b)
	defer db.Close()

	// write 1 million
	threshold := 1000000
	value4KB := bitcask.GenNKBytes(4)

	batchSize := 50
	batch := db.NewWriteBatch()
	for i := 1; i <= threshold; i++ {
		err := batch.Set(newKey(i, threshold), value4KB)
		assert.Nil(b, err)

		if i%batchSize == 0 {
			err = batch.Flush()
			assert.Nil(b, err)
			batch = db.NewWriteBatch()
		}
	}

	b.Run("read4K", func(b *testing.B) {
		benchmarkReadWithBadger(b, db, threshold)
	})
}

func benchmarkReadWithBadger(b *testing.B, db *badger.DB, threshold int) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := db.View(func(txn *badger.Txn) error {
				_, err := txn.Get(randomKey(threshold))
				return err
			})
			assert.Nil(b, err)
		}
	})

	reportRss(b)
}

func BenchmarkReadWithLevelDB(b *testing.B) {
	db := getLevelDB(b)
	defer db.Close()

	// write 1 million
	threshold := 1000000
	value4KB := bitcask.GenNKBytes(4)
	batchSize := 50
	batch := new(leveldb.Batch)
	for i := 1; i <= threshold; i++ {
		batch.Put(newKey(i, threshold), value4KB)

		if i%batchSize == 0 {
			err := db.Write(batch, nil)
			assert.Nil(b, err)
			batch.Reset()
		}
	}

	b.Run("read4K", func(b *testing.B) {
		benchmarkReadWithLevelDB(b, db, threshold)
	})
}

func benchmarkReadWithLevelDB(b *testing.B, db *leveldb.DB, threshold int) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := db.Get(randomKey(threshold), nil)
			assert.Nil(b, err)
		}
	})

	reportRss(b)
}

func BenchmarkReadWithBoltDB(b *testing.B) {
	db := getBoltDB(b)
	defer db.Close()

	bucketName := []byte("benchmark")

	// write 1 million
	threshold := 1000000
	value4KB := bitcask.GenNKBytes(4)
	for i := 1; i <= threshold; i += 50 {
		err := db.Update(func(txn *bolt.Tx) error {
			bucket := txn.Bucket(bucketName)
			for j := 0; j < 50; j++ {
				err := bucket.Put(newKey(i+j, threshold), value4KB)
				assert.Nil(b, err)
			}
			return nil
		})
		assert.Nil(b, err)
	}

	b.Run("read4K", func(b *testing.B) {
		benchmarkReadWithBoltDB(b, db, bucketName, threshold)
	})
}

func benchmarkReadWithBoltDB(b *testing.B, db *bolt.DB, bucketName []byte, threshold int) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := db.View(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(bucketName)
				v := bucket.Get(randomKey(threshold))
				assert.NotNil(b, v)
				return nil
			})
			assert.Nil(b, err)
		}
	})

	reportRss(b)
}

func BenchmarkWriteWithBitcaskDB(b *testing.B) {
	db := getBitcaskDB(b)
	defer db.Close()

	b.Run("write4K", func(b *testing.B) {
		benchmarkWriteWithBitcaskDB(b, db)
	})
}

func benchmarkWriteWithBitcaskDB(b *testing.B, db bitcask.DB) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	// repeat write 10 millions
	threshold := 10000000
	meta := bitcask.NewMeta(nil)
	value4KB := bitcask.GenNKBytes(4)
	opts := &bitcask.WriteOptions{}
	batchSize := 50

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		iteration := 1
		batch := bitcask.NewBatch()
		for pb.Next() {
			batch.Put(nil, newKey(iteration, threshold), value4KB, meta)

			if iteration%batchSize == 0 {
				err := db.Write(batch, opts)
				assert.Nil(b, err)
				batch.Clear()
			}

			iteration++
		}
	})

	reportRss(b)
}

func BenchmarkWriteWithLevelDB(b *testing.B) {
	db := getLevelDB(b)
	defer db.Close()

	b.Run("write4K", func(b *testing.B) {
		benchmarkWriteWithLevelDB(b, db)
	})
}

func benchmarkWriteWithLevelDB(b *testing.B, db *leveldb.DB) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	// repeat write 10 millions
	threshold := 10000000
	value4KB := bitcask.GenNKBytes(4)
	batchSize := 50

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		iteration := 1
		batch := new(leveldb.Batch)
		for pb.Next() {
			batch.Put(newKey(iteration, threshold), value4KB)

			if iteration%batchSize == 0 {
				err := db.Write(batch, nil)
				assert.Nil(b, err)
				batch.Reset()
			}

			iteration++
		}
	})

	reportRss(b)
}

func BenchmarkWriteWithBadger(b *testing.B) {
	db := getBadgerDB(b)
	defer db.Close()

	b.Run("write4K", func(b *testing.B) {
		benchmarkWriteWithBadger(b, db)
	})
}

func benchmarkWriteWithBadger(b *testing.B, db *badger.DB) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	// repeat write 10 millions
	threshold := 10000000
	value4KB := bitcask.GenNKBytes(4)
	batchSize := 50

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		iteration := 1
		batch := db.NewWriteBatch()
		for pb.Next() {
			err := batch.Set(newKey(iteration, threshold), value4KB)
			assert.Nil(b, err)

			if iteration%batchSize == 0 {
				assert.Nil(b, batch.Flush())
				batch = db.NewWriteBatch()
			}

			iteration++
		}
	})

	reportRss(b)
}

func BenchmarkWriteWithBoltDB(b *testing.B) {
	db := getBoltDB(b)
	defer db.Close()

	b.Run("write4K", func(b *testing.B) {
		benchmarkWriteWithBoltDB(b, db)
	})
}

func benchmarkWriteWithBoltDB(b *testing.B, db *bolt.DB) {
	rssSamples = make([]uint64, 0, 120)

	stopCh = make(chan struct{})
	defer close(stopCh)

	go sampleRSS(b)

	// repeat write 10 millions
	threshold := 10000000
	value4KB := bitcask.GenNKBytes(4)
	bucketName := []byte("benchmark")
	batchSize := 50

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		iteration := 1
		for pb.Next() {
			if iteration%batchSize == 0 {
				err := db.Update(func(txn *bolt.Tx) error {
					bucket := txn.Bucket(bucketName)
					for i := 1; i <= batchSize; i++ {
						err := bucket.Put(newKey(iteration-batchSize+i, threshold), value4KB)
						assert.Nil(b, err)
					}
					return nil
				})
				assert.Nil(b, err)
			}

			iteration++
		}
	})

	reportRss(b)
}

func getActualDiskUsage(path string) int64 {
	cmd := exec.Command("du", "-sb", path)

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0
	}

	parts := strings.Fields(out.String())
	if len(parts) < 1 {
		return 0
	}

	size, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}

	return size
}

func reportDiskUsage(b *testing.B, path string) {
	usage := getActualDiskUsage(path)
	b.ReportMetric(float64(usage)/1024/1024/1024, "DiskUsage(GB)")
}

func BenchmarkDiskUsageWithBitcaskDB(b *testing.B) {
	db := getBitcaskDB(b)
	defer db.Close()

	// write 1 million
	threshold := 1000000
	meta := bitcask.NewMeta(nil)
	value4KB := bitcask.GenNKBytes(4)
	opts := &bitcask.WriteOptions{}

	batchSize := 50
	batch := bitcask.NewBatch()
	for i := 1; i <= threshold; i++ {
		batch.Put(nil, newKey(i, threshold), value4KB, meta)

		if i%batchSize == 0 {
			err := db.Write(batch, opts)
			assert.Nil(b, err)
			batch.Clear()
		}
	}

	b.Run("1 million", func(b *testing.B) {
		reportDiskUsage(b, "./bitcaskDB")
	})
}

func BenchmarkDiskUsageWithLevelDB(b *testing.B) {
	db := getLevelDB(b)
	defer db.Close()

	// write 1 million
	threshold := 1000000
	value4KB := bitcask.GenNKBytes(4)
	batchSize := 50
	batch := new(leveldb.Batch)
	for i := 1; i <= threshold; i++ {
		batch.Put(newKey(i, threshold), value4KB)

		if i%batchSize == 0 {
			err := db.Write(batch, nil)
			assert.Nil(b, err)
			batch.Reset()
		}
	}

	b.Run("1 million", func(b *testing.B) {
		reportDiskUsage(b, "./leveldb")
	})
}

func BenchmarkDiskUsageWithBadger(b *testing.B) {
	db := getBadgerDB(b)
	defer db.Close()

	// write 1 million
	threshold := 1000000
	value4KB := bitcask.GenNKBytes(4)

	batchSize := 50
	batch := db.NewWriteBatch()
	for i := 1; i <= threshold; i++ {
		err := batch.Set(newKey(i, threshold), value4KB)
		assert.Nil(b, err)

		if i%batchSize == 0 {
			err = batch.Flush()
			assert.Nil(b, err)
			batch = db.NewWriteBatch()
		}
	}

	b.Run("1 million", func(b *testing.B) {
		reportDiskUsage(b, "./badger")
	})
}

func BenchmarkDiskUsageWithBoltDB(b *testing.B) {
	db := getBoltDB(b)
	defer db.Close()

	bucketName := []byte("benchmark")

	// write 1 million
	threshold := 1000000
	value4KB := bitcask.GenNKBytes(4)
	for i := 1; i <= threshold; i += 50 {
		err := db.Update(func(txn *bolt.Tx) error {
			bucket := txn.Bucket(bucketName)
			for j := 0; j < 50; j++ {
				err := bucket.Put(newKey(i+j, threshold), value4KB)
				assert.Nil(b, err)
			}
			return nil
		})
		assert.Nil(b, err)
	}

	b.Run("1 million", func(b *testing.B) {
		reportDiskUsage(b, "./bolt")
	})
}
