package project

import (
	"errors"
	"testing"
	"time"
)

func TestResourceCursorRoundTrip(t *testing.T) {
	wantTime := time.Date(2026, 7, 13, 9, 30, 0, 123, time.UTC)
	encoded := encodeResourceCursor(wantTime, 42)
	decoded, err := decodeResourceCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !decoded.Time.Equal(wantTime) || decoded.ID != 42 {
		t.Fatalf("decoded cursor = %+v", decoded)
	}
}

func TestResourceCursorRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"not-base64", "e30", encodeResourceCursor(time.Time{}, 0)} {
		if _, err := decodeResourceCursor(value); !errors.Is(err, ErrInvalidResourceCursor) {
			t.Fatalf("decodeResourceCursor(%q) error = %v", value, err)
		}
	}
}

func TestNormalizeResourcePageOptions(t *testing.T) {
	options := normalizeResourcePageOptions(ResourcePageOptions{ProjectKey: " p ", Status: " active ", Limit: 500})
	if options.ProjectKey != "p" || options.Status != "active" || options.Limit != 200 {
		t.Fatalf("normalized options = %+v", options)
	}
	if got := normalizeResourcePageOptions(ResourcePageOptions{}).Limit; got != 50 {
		t.Fatalf("default limit = %d", got)
	}
}
