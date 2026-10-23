package db

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestHashUploadTokenLegacyCanonicalization(t *testing.T) {
	canonical := "123e4567-e89b-12d3-a456-426614174000"
	sum := sha256.Sum256([]byte(canonical))
	expected := hex.EncodeToString(sum[:])
	for _, raw := range []string{canonical, strings.ToUpper(canonical), "123e4567e89b12d3a456426614174000", "urn:uuid:" + canonical, "{" + canonical + "}"} {
		got, err := HashUploadToken(raw)
		if err != nil || got != expected {
			t.Errorf("%q: %q %v", raw, got, err)
		}
	}
}
func TestHashUploadTokenRejectsMalformedSecrets(t *testing.T) {
	for _, raw := range []string{"", "not-a-token", "hl_upload_", "hl_upload_" + strings.Repeat("A", 42), "hl_upload_" + strings.Repeat("A", 44), "hl_upload_" + strings.Repeat("/", 43), "hl_upload_" + strings.Repeat("A", 42) + "B"} {
		if _, err := HashUploadToken(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}
