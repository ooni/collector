package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ooni/collector/collector/report"
)

// GinStorageMiddleware a database aware middleware.
// It will set the Storage property, that can be accessed via:
// storage := c.MustGet("Storage").(*report.Storage)
type GinStorageMiddleware struct {
	Storage *report.Storage
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
func InitStorageMiddleware(s *report.Storage) (*GinStorageMiddleware, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}
	return &GinStorageMiddleware{Storage: s}, nil
}
