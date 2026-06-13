package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// ValidateSyncKey rejects sync requests without a valid X-Sync-Key header.
// Skipped entirely if CLOUD_API_KEY env var is not set.
func ValidateSyncKey() gin.HandlerFunc {
	key := os.Getenv("CLOUD_API_KEY")
	if key == "" {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return func(c *gin.Context) {
		if c.GetHeader("X-Sync-Key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid sync key"})
			return
		}
		c.Next()
	}
}
