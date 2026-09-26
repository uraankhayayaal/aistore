package qdrant

import (
	"regexp"
	"testing"

	"github.com/google/uuid"
)

// TestUUIDGenerationFormat проверяет, что генерируемый UUID соответствует формату v4.
func TestUUIDGenerationFormat(t *testing.T) {
	re := regexp.MustCompile(`^k-b-c-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for i := 0; i < 100; i++ {
		n := uuid.New()
		newID := "k-b-c-" + n.String()
		if !re.MatchString(newID) {
			t.Errorf("UUID %q не соответсвует формату v4", newID)
		}
	}
}

// TestUUIDFormatV4Valid проверяет валидность формата UUID v4.
func TestUUIDFormatV4Valid(t *testing.T) {
	// UUID v4:xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx, где y = 8, 9, a, или b
	if !validateUUIDV4(uuid.New().String()) {
		t.Error("uuid.New() должен генерировать UUID v4, но validation прошёл неудачно")
	}

	prefixUUID := "k-b-c-" + uuid.New().String()
	suffix := prefixUUID[len("k-b-c-"):]
	suffixParts := validateUUIDV4(suffix)
	if !suffixParts {
		t.Errorf("Предфиксированный UUID %q должен пройти валидацию UUID v4", prefixUUID)
	}
}

// validateUUIDV4 проверяет, что строка содержит UUID формата v4.
func validateUUIDV4(s string) bool {
	if len(s) != 36 {
		return false
	}

	for i, r := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}

	// Проверяем version = 4 в position 14 (0-based: 14)
	if s[14] != '4' {
		return false
	}

	// Проверяем variant в position 19 — один из 8, 9, a, b
	switch s[19] {
	case '8', '9', 'a', 'b':
		return true
	default:
		return false
	}
}

// TestConsecutiveUUIDsUnique проверяет, что последовательные UUID уникальны.
func TestConsecutiveUUIDsUnique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		n := uuid.New().String()
		if ids[n] {
			t.Errorf("UUID %q не уникален", n)
		}
		ids[n] = true
	}
}
