package db

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestHashSessionTokenLegacyCanonicalization(t *testing.T) {
	canonical := "123e4567-e89b-12d3-a456-426614174000"
	sum := sha256.Sum256([]byte(canonical))
	expected := hex.EncodeToString(sum[:])
	for _, raw := range []string{canonical, strings.ToUpper(canonical), "123e4567e89b12d3a456426614174000", "urn:uuid:" + canonical, "{" + canonical + "}"} {
		got, err := HashSessionToken(raw)
		if err != nil || got != expected {
			t.Errorf("%q: %q %v", raw, got, err)
		}
	}
}
func TestHashSessionTokenRejectsMalformedSecrets(t *testing.T) {
	for _, raw := range []string{"", "invalid", "hl_session_", "hl_session_" + strings.Repeat("A", 42), "hl_session_" + strings.Repeat("A", 44), "hl_session_" + strings.Repeat("/", 43), "hl_session_" + strings.Repeat("A", 42) + "B", "hl_upload_" + strings.Repeat("A", 43)} {
		if _, err := HashSessionToken(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}
