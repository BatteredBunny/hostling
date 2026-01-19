package internal

import (
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

func parseToken(rawToken string) (uuid.UUID, error) {
	return uuid.Parse(rawToken)
}

func formatTimeDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return humanize.Time(t)
}

func mimeIsImage(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

func mimeIsVideo(mimeType string) bool {
	return strings.HasPrefix(mimeType, "video/")
}

func mimeIsAudio(mimeType string) bool {
	return strings.HasPrefix(mimeType, "audio/")
}

func humanizeBytes(size uint) string {
	return humanize.Bytes(uint64(size))
}

func Sum[T any](slice []T, getValue func(T) int) int {
	sum := 0
	for _, item := range slice {
		sum += getValue(item)
	}
	return sum
}
