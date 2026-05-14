package internal

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

var fuzzLinkingKey = []byte("00000000000000000000000000000000")

func FuzzParseLinkingValue(f *testing.F) {
	now := time.Unix(1_700_000_000, 0)

	// Okay
	validNotExpired := signLinkingValue(42, fuzzLinkingKey, now.Add(time.Hour))
	f.Add(validNotExpired)

	// Different key
	wrongKey := signLinkingValue(42, []byte("wrong-key-wrong-key-wrong-key-aa"), now.Add(time.Hour))
	f.Add(wrongKey)

	// Expired
	expired := signLinkingValue(42, fuzzLinkingKey, now.Add(-time.Hour))
	f.Add(expired)

	// Pathological seeds.
	for _, s := range []string{
		"",
		".",
		"..",
		"...",
		"a.b.c",
		"1.2.3",
		"1.\x00.\x00",
		"-1.-1.-1",
		"99999999999999999999999999999.0.x",
		string(make([]byte, 1024)),
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		// Must never panic.
		id, err := parseLinkingValue(raw, fuzzLinkingKey, now)

		if err == nil {
			// Only the freshly-signed seed should ever land here. If a
			// random input validated, the HMAC was forged or the parser
			// accepted unsigned input.
			expected, expectedErr := parseLinkingValue(validNotExpired, fuzzLinkingKey, now)
			if expectedErr != nil || raw != validNotExpired || id != expected {
				t.Fatalf("parseLinkingValue accepted unexpected input: raw=%q id=%d", raw, id)
			}
		} else if !errors.Is(err, ErrInvalidLinkingCookie) {
			t.Fatalf("unexpected error type for raw=%q: %v", raw, err)
		}
	})
}

func FuzzSignParseRoundtrip(f *testing.F) {
	f.Add(uint64(0), int64(60))
	f.Add(uint64(1), int64(3600))
	f.Add(uint64(^uint32(0)), int64(86400))
	f.Add(uint64(42), int64(1))

	f.Fuzz(func(t *testing.T, id uint64, secondsAhead int64) {
		// Constrain expiry to the strictly-future side; signLinkingValue
		// itself doesn't validate this, but parseLinkingValue rejects past.
		if secondsAhead <= 0 {
			secondsAhead = 1
		}
		now := time.Unix(1_700_000_000, 0)
		expiry := now.Add(time.Duration(secondsAhead) * time.Second)

		signed := signLinkingValue(uint(id), fuzzLinkingKey, expiry)
		got, err := parseLinkingValue(signed, fuzzLinkingKey, now)
		if err != nil {
			t.Fatalf("roundtrip failed for id=%d exp=%v: %v", id, expiry, err)
		}
		if uint64(got) != uint64(uint(id)) {
			t.Fatalf("roundtrip id mismatch: signed %d, parsed %d", id, got)
		}
	})
}

// make sure partial keys arent accepted
func FuzzBitFlipRejection(f *testing.F) {
	now := time.Unix(1_700_000_000, 0)
	const knownID uint = 7
	valid := signLinkingValue(knownID, fuzzLinkingKey, now.Add(time.Hour))

	f.Add(0, byte(0xFF))
	f.Add(len(valid)-1, byte(0x01))
	f.Add(len(valid)/2, byte(0x80))

	f.Fuzz(func(t *testing.T, idx int, mutation byte) {
		if idx < 0 || idx >= len(valid) || mutation == 0 {
			t.Skip()
		}
		b := []byte(valid)
		original := b[idx]
		b[idx] ^= mutation
		if b[idx] == original {
			t.Skip()
		}

		id, err := parseLinkingValue(string(b), fuzzLinkingKey, now)
		if err == nil && id != knownID {
			t.Fatalf("mutated value validated as different id (idx=%d, ^=0x%02x): got id=%d, value=%q",
				idx, mutation, id, b)
		}
	})
}

// every freshly generated key with a freshly signed value should roundtrip with itself and reject with any other key.
func FuzzKeyIsolation(f *testing.F) {
	f.Add(uint64(1), int64(60))
	f.Add(uint64(9999), int64(3600))

	f.Fuzz(func(t *testing.T, id uint64, secondsAhead int64) {
		if secondsAhead <= 0 {
			secondsAhead = 1
		}
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)
		_, _ = rand.Read(key1)
		_, _ = rand.Read(key2)

		now := time.Unix(1_700_000_000, 0)
		expiry := now.Add(time.Duration(secondsAhead) * time.Second)

		signed := signLinkingValue(uint(id), key1, expiry)
		if _, err := parseLinkingValue(signed, key1, now); err != nil {
			t.Fatalf("self-key parse failed: %v", err)
		}
		if _, err := parseLinkingValue(signed, key2, now); err == nil {
			t.Fatalf("cross-key parse accepted; HMAC isolation broken")
		}
	})
}
