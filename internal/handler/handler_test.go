package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/BuzzLyutic/03.12.2025/internal/model"
	"github.com/BuzzLyutic/03.12.2025/internal/pdf"
	"github.com/BuzzLyutic/03.12.2025/internal/service"
	"github.com/BuzzLyutic/03.12.2025/internal/storage"
)

func setupTestHandler(t *testing.T) (*Handler, func()) {
	tmpFile := "test_handler_storage.json"
	store, err := storage.New(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	checker := service.NewChecker(5 * time.Second)
	pdfGen := pdf.NewGenerator()
	h := New(checker, store, pdfGen)

	cleanup := func() {
		os.Remove(tmpFile)
	}

	return h, cleanup
}

func TestHandler_CheckLinks(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"links": ["google.com"]}`
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.CheckLinks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp model.CheckLinksResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.LinksNum != 1 {
		t.Errorf("Expected links_num 1, got %d", resp.LinksNum)
	}

	if _, ok := resp.Links["google.com"]; !ok {
		t.Error("Expected google.com in response")
	}
}

func TestHandler_CheckLinks_InvalidMethod(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/check", nil)
	w := httptest.NewRecorder()

	h.CheckLinks(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandler_CheckLinks_EmptyLinks(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"links": []}`
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.CheckLinks(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandler_GetReport(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	// Сначала добавить ссылки
	body := `{"links": ["google.com"]}`
	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CheckLinks(w, req)

	// Теперь запросить отчёт
	reportBody := `{"links_list": [1]}`
	req = httptest.NewRequest(http.MethodPost, "/report", bytes.NewBufferString(reportBody))
	w = httptest.NewRecorder()

	h.GetReport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("Expected content-type application/pdf, got %s", contentType)
	}
}

func TestHandler_GetReport_NotFound(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	body := `{"links_list": [999]}`
	req := httptest.NewRequest(http.MethodPost, "/report", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.GetReport(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandler_HealthCheck(t *testing.T) {
	h, cleanup := setupTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandler_GracefulShutdown(t *testing.T) {
    h, cleanup := setupTestHandler(t)
    defer cleanup()

    // Симуляция активной задачи
    h.activeJobs.Add(1)

    // Начало shutdown
    h.StartShutdown()

    // Новый запрос должен вернуть 202
    body := `{"links": ["google.com"]}`
    req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewBufferString(body))
    w := httptest.NewRecorder()

    h.CheckLinks(w, req)

    if w.Code != http.StatusAccepted {
        t.Errorf("Expected 202 during shutdown, got %d", w.Code)
    }

    // Проверка что задача сохранена как pending
    task := h.storage.GetAndClearPendingTasks()
    // Должна быть минимум 1 pending задача
    if len(task) < 1 {
		t.Errorf("Expected at least 1 pending task")
	}
    h.activeJobs.Done()
}


func TestHandler_ProcessPendingTasksOnRestart(t *testing.T) {
    tmpFile := "test_restart_storage.json"
    defer os.Remove(tmpFile)

    // Симуляция shutdown с pending задачей
    store1, _ := storage.New(tmpFile)
    store1.AddPendingTask(model.PendingTask{
        ID:    1,
        Links: []string{"google.com"},
    })
    store1.Save()

    // "Перезапуск" — новый экземпляр
    store2, _ := storage.New(tmpFile)
    checker := service.NewChecker(5 * time.Second)
    pdfGen := pdf.NewGenerator()
    h := New(checker, store2, pdfGen)

    // Обработка pending задачи (как при реальном запуске)
    h.ProcessPendingTasks(context.Background())

    // Проверка что задача обработана и сохранена как результат
    linkSet, err := store2.GetLinkSet(1)
    if err != nil {
        t.Fatalf("Expected link set with ID 1, got error: %v", err)
    }

    if _, ok := linkSet.Links["google.com"]; !ok {
        t.Error("Expected google.com in processed link set")
    }

    // Проверка что pending очищен
    pending := store2.GetAndClearPendingTasks()
    if len(pending) != 0 {
        t.Errorf("Expected 0 pending tasks after processing, got %d", len(pending))
    }
}
