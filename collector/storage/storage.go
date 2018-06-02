package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/dgraph-io/badger"
)

// ReportMetadata contains metadata about the report
type ReportMetadata struct {
	ReportID        string
	ProbeASN        string
	ProbeCC         string
	Platform        string
	TestName        string
	SoftwareName    string
	SoftwareVersion string
	ReportFilePath  string
	CreationTime    time.Time
	LastUpdateTime  time.Time
	EntryCount      int64
	Closed          bool
}

// By default we will expire the report metadata after 30 days
const (
	reportExpiryDuration      = 24 * 30 * time.Hour
	garbageCollectionInterval = 10 * time.Minute
	discardRatio              = 0.5
)

// New func implements the storage interface for gorush (https://github.com/appleboy/gorush)
func New(dir string) *Storage {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	return &Storage{
		db:   nil,
		opts: opts,
	}
}

// Storage interface implementation for badger
type Storage struct {
	opts       badger.Options
	db         *badger.DB
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Init checks that the store is usable
func (s *Storage) Init() error {
	db, err := badger.Open(s.opts)
	if err != nil {
		return err
	}
	s.db = db
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	go s.runGarbageCollection()
	return nil
}

// SetReport writes the report metadata to the store
func (s *Storage) SetReport(m *ReportMetadata) error {
	var err error
	err = s.db.Update(func(txn *badger.Txn) error {
		var value []byte
		if value, err = json.Marshal(m); err != nil {
			return err
		}
		err = txn.SetWithTTL([]byte(fmt.Sprintf("report/%s", m.ReportID)), value, reportExpiryDuration)
		return err
	})
	return err
}

// ErrReportNotFound indicates no report with the given id could be found
var ErrReportNotFound = errors.New("Report not found")

// GetReport returns a report based on it's reportID
func (s *Storage) GetReport(reportID string) (*ReportMetadata, error) {
	var (
		meta ReportMetadata
		err  error
	)

	err = s.db.View(func(txn *badger.Txn) error {
		var (
			item *badger.Item
			val  []byte
		)
		if item, err = txn.Get([]byte(fmt.Sprintf("report/%s", reportID))); err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrReportNotFound
			}
			return err
		}
		if val, err = item.Value(); err != nil {
			return err
		}
		if err = json.Unmarshal(val, &meta); err != nil {
			return err
		}
		return nil
	})
	return &meta, err
}

// ListReports returns all the reports in the store
func (s *Storage) ListReports() ([]*ReportMetadata, error) {
	var (
		reports []*ReportMetadata
		err     error
	)

	err = s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := []byte("report/")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var val []byte
			var meta ReportMetadata
			item := it.Item()
			if val, err = item.Value(); err != nil {
				return err
			}
			if err = json.Unmarshal(val, &meta); err != nil {
				return err
			}
			reports = append(reports, &meta)
		}
		return nil
	})
	return reports, err
}

// Close the database cleanly
func (s *Storage) Close() error {
	// cancel (db) context
	s.cancelFunc()
	// close db
	err := s.db.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) runGarbageCollection() {
	ticker := time.NewTicker(garbageCollectionInterval)
	for {
		select {
		case <-ticker.C:
			err := s.db.RunValueLogGC(discardRatio)
			if err != nil {
				// don't report error when gc didn't result in any cleanup
				if err == badger.ErrNoRewrite {
					log.Debugf("Badger GC: %v", err)
				} else {
					log.Errorf("Badger GC failed: %v", err)
				}
			}
		case <-s.ctx.Done():
			return
		}
	}

}
