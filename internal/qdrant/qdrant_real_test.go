package qdrant

import (
	"regexp"
	"testing"

	"github.com/google/uuid"
)

// TestUuidFormatValidUuidv4 проверяет что сгенерированные UUID имеют корректный формат UUID v4.
func TestUuidFormatValidUuidv4(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := uuid.New().String()
		if !re.MatchString(id) {
			t.Errorf("UUID не формат UUID v4: %s", id)
		}
		if seen[id] {
			t.Errorf("Повтор UUID: %s", id)
		}
		seen[id] = true
	}
}

// TestGeneratePointIdFormat проверяет формат строки точки: "k-b-c-{uuid}".
func TestGeneratePointIdFormat(t *testing.T) {
	pointRe := regexp.MustCompile(`^k-b-c-[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		pointID := generatePointID()
		if !pointRe.MatchString(pointID) {
			t.Errorf("pointID не формат к-b-c-uuid: %s", pointID)
		}
		if seen[pointID] {
			t.Errorf("Повтор pointID: %s", pointID)
		}
		seen[pointID] = true
	}
}

// TestUpsertChunkUniqueIds проверяет что каждый вызов generatePointID возвращает уникальный ID.
func TestUpsertChunkUniqueIds(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := generatePointID()
		if seen[id] {
			t.Errorf("ID не уникален после %d итераций: %s", i, id)
			break
		}
		seen[id] = true
	}
}
