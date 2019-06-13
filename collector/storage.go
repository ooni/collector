package collector

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/gin-gonic/gin"
	"golang.org/x/sys/unix"
)

// NewStorage creates a new storage backend
func NewStorage(reportDir string) (*Storage, error) {
	return &Storage{
		reportDir:     reportDir,
		activeReports: make(map[string]*ActiveReport),
	}, nil
}

// Storage interface implementation
type Storage struct {
	reportDir     string
	activeReports map[string]*ActiveReport
}

// IncomingDir is where report files are stored while they are active
func (s *Storage) IncomingDir() string {
	return filepath.Join(s.reportDir, "incoming")
}

// SyncDir is where reports are stored when they are ready to be synched (or closed)
func (s *Storage) SyncDir() string {
	return filepath.Join(s.reportDir, "sync")
}

// Init checks that the store is usable
func (s *Storage) Init() error {
	for _, path := range []string{s.SyncDir(), s.IncomingDir()} {
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0770); err != nil {
				return fmt.Errorf("Failed to create: %s", path)
			}
		} else if !stat.IsDir() || unix.Access(path, unix.W_OK) != nil {
			return fmt.Errorf("Wrong permissions for report_dir: %s", path)
		}
	}
	return nil
}

// CreateReportFile creates a file to store a set of measurements
func (s *Storage) CreateReportFile(activeReport *ActiveReport) error {
	path := filepath.Join(s.IncomingDir(), activeReport.IncomingFilename())
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0770)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	activeReport.Path = path
	return nil
}

// WriteToReportFile will append to an active report file
func (s *Storage) WriteToReportFile(activeReport *ActiveReport, data []byte) error {
	path := filepath.Join(s.IncomingDir(), activeReport.IncomingFilename())
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0750)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

// CloseReportFile wll move the report file from incoming into the sync directory
func (s *Storage) CloseReportFile(activeReport *ActiveReport) error {
	srcPath := filepath.Join(s.IncomingDir(), activeReport.IncomingFilename())
	fi, err := os.Stat(srcPath)
	if err != nil {
		return err
	}
	// Empty expired report files are simply deleted
	if fi.Size() == 0 {
		return os.Remove(srcPath)
	}

	reportFilename, err := activeReport.SyncFilename()
	if err != nil {
		log.WithError(err).Error("failed to generate filename")
		return err
	}
	dstPath := filepath.Join(s.SyncDir(), reportFilename)
	return os.Rename(srcPath, dstPath)
}

// Close the storage cleanly
func (s *Storage) Close() error {
	return nil
}

// GinStorageMiddleware a database aware middleware.
// It will set the Storage property, that can be accessed via:
// storage := c.MustGet("Storage").(*Storage)
type GinStorageMiddleware struct {
	Storage *Storage
}

// MiddlewareFunc this is what you register as the middleware, like this:
// router.Use(storageMiddleware.MiddlewareFunc())
func (mw *GinStorageMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("Storage", mw.Storage)
		c.Next()
	}
}

// InitStorageMiddleware create the middleware that injects the storage backend
func InitStorageMiddleware(s *Storage) (*GinStorageMiddleware, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}
	return &GinStorageMiddleware{Storage: s}, nil
}
