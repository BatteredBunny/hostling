package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
)

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

func formatContentDisposition(disposition, filename string) string {
	var ascii strings.Builder
	for _, r := range filename {
		if r < ' ' || r > '~' || r == '"' || r == '\\' {
			ascii.WriteByte('_')

			continue
		}
		ascii.WriteRune(r)
	}

	return fmt.Sprintf(
		`%s; filename="%s"; filename*=UTF-8''%s`,
		disposition,
		ascii.String(),
		encodeRFC5987(filename),
	)
}

// Percent-encodes everything outside RFC 5987 attr-char, url.PathEscape leaves chars like = and : that aren't allowed
func encodeRFC5987(s string) string {
	const upperhex = "0123456789ABCDEF"

	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || strings.IndexByte("!#$&+-.^_`|~", c) >= 0 {
			b.WriteByte(c)

			continue
		}
		b.WriteByte('%')
		b.WriteByte(upperhex[c>>4])
		b.WriteByte(upperhex[c&15])
	}

	return b.String()
}

// Dangerous mime types that we force download for to prevent potential malicious file from rendering or executing
var dangerousInlineMimes = map[string]bool{
	"text/html":              true,
	"application/xhtml+xml":  true,
	"application/xml":        true,
	"text/xml":               true,
	"image/svg+xml":          true,
	"application/javascript": true,
	"text/javascript":        true,
}

func uploadDisposition(mimeType string) string {
	base, _, _ := strings.Cut(mimeType, ";")
	if dangerousInlineMimes[strings.TrimSpace(base)] {
		return "attachment"
	}

	return "inline"
}

func setUploadServeHeaders(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header(
		"Content-Security-Policy",
		"sandbox; default-src 'none'; img-src 'self'; media-src 'self'; style-src 'unsafe-inline'",
	)
	c.Header("Referrer-Policy", "no-referrer")
}
