package middleware

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"bugyou-backend/internal/config"

	"github.com/gin-gonic/gin"
)

func SaveOptionalAttachment(c *gin.Context) (string, bool) {
	file, err := c.FormFile("attachment")
	if err != nil {
		return "", true
	}

	extension := filepath.Ext(file.Filename)
	base := strings.TrimSuffix(filepath.Base(file.Filename), extension)
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), sanitizeFilename(base), extension)
	path := filepath.Join(config.Values.UploadDir, filename)

	if err := c.SaveUploadedFile(file, path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "attachment upload failed"})
		return "", false
	}

	return "/" + path, true
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, value)
	if value == "" {
		return "attachment"
	}

	return value
}
