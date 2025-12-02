package storage

import (
	"os"
	"testing"

	"github.com/BuzzLyutic/03.12.2025/internal/model"
)

func TestStorage_AddAndGet(t *testing.T) {
	tmpFile := "test_storage.json"
	defer os.Remove(tmpFile)

	s, err := New(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	links := map[string]string{
		"google.com": "available",
		"invalid.xx": "not available",
	}

	id := s.NextID()
	s.AddLinkSet(id, links)

	got, err := s.GetLinkSet(id)
	if err != nil {
		t.Fatalf("Failed to get link set: %v", err)
	}

	if got.ID != id {
		t.Errorf("Expected ID %d, got %d", id, got.ID)
	}

	if len(got.Links) != len(links) {
		t.Errorf("Expected %d links, got %d", len(links), len(got.Links))
	}
}

func TestStorage_Persistence(t *testing.T) {
	tmpFile := "test_persistence.json"
	defer os.Remove(tmpFile)

	// Создание и сохранение
	s1, _ := New(tmpFile)
	id := s1.NextID()
	s1.AddLinkSet(id, map[string]string{"test.com": "available"})
	s1.Save()

	// Загрузка в новый экземпляр
	s2, err := New(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load storage: %v", err)
	}

	got, err := s2.GetLinkSet(id)
	if err != nil {
		t.Fatalf("Failed to get link set after reload: %v", err)
	}

	if got.Links["test.com"] != "available" {
		t.Errorf("Expected 'available', got '%s'", got.Links["test.com"])
	}
}

func TestStorage_PendingTasks(t *testing.T) {
	tmpFile := "test_pending.json"
	defer os.Remove(tmpFile)

	s, _ := New(tmpFile)

	task := model.PendingTask{
		ID:    1,
		Links: []string{"google.com"},
	}
	s.AddPendingTask(task)
	s.Save()

	// Загрузка заново
	s2, _ := New(tmpFile)
	tasks := s2.GetAndClearPendingTasks()

	if len(tasks) != 1 {
		t.Fatalf("Expected 1 pending task, got %d", len(tasks))
	}

	if tasks[0].ID != 1 {
		t.Errorf("Expected task ID 1, got %d", tasks[0].ID)
	}
}
