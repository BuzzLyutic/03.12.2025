package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/BuzzLyutic/03.12.2025/internal/model"
	"github.com/BuzzLyutic/03.12.2025/internal/pdf"
	"github.com/BuzzLyutic/03.12.2025/internal/service"
	"github.com/BuzzLyutic/03.12.2025/internal/storage"
)

type Handler struct {
	checker    *service.Checker
	storage    *storage.Storage
	pdfGen     *pdf.Generator
	activeJobs sync.WaitGroup
	shutdown   bool
	mu         sync.RWMutex
}

func New(checker *service.Checker, storage *storage.Storage, pdfGen *pdf.Generator) *Handler {
	return &Handler{
		checker: checker,
		storage: storage,
		pdfGen:  pdfGen,
	}
}

// CheckLinks обрабатывает запрос на проверку ссылок
func (h *Handler) CheckLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.CheckLinksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Links) == 0 {
		http.Error(w, "No links provided", http.StatusBadRequest)
		return
	}

	// Проверяем не в процессе ли shutdown
	h.mu.RLock()
	isShutdown := h.shutdown
	h.mu.RUnlock()

	// Получаем ID заранее для возможного сохранения
	id := h.storage.NextID()

	if isShutdown {
		// Сохраняем задачу как pending и возвращаем
		h.storage.AddPendingTask(model.PendingTask{
			ID:    id,
			Links: req.Links,
		})
		h.storage.Save()

		// Возвращаем ответ что задача принята
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Task queued for processing after restart",
			"links_num": id,
		})
		return
	}

	// Увеличиваем счётчик активных задач
	h.activeJobs.Add(1)
	defer h.activeJobs.Done()

	// Добавляем как pending на случай crash
	h.storage.AddPendingTask(model.PendingTask{
		ID:    id,
		Links: req.Links,
	})

	// Проверяем ссылки
	results := h.checker.CheckLinks(r.Context(), req.Links)

	// Удаляем из pending и добавляем результат
	h.storage.RemovePendingTask(id)
	h.storage.AddLinkSet(id, results)

	// Сохраняем состояние
	if err := h.storage.Save(); err != nil {
		log.Printf("Failed to save storage: %v", err)
	}

	resp := model.CheckLinksResponse{
		Links:    results,
		LinksNum: id,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetReport обрабатывает запрос на получение PDF отчёта
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.GetReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.LinksList) == 0 {
		http.Error(w, "No links_list provided", http.StatusBadRequest)
		return
	}

	linkSets := h.storage.GetLinkSets(req.LinksList)
	if len(linkSets) == 0 {
		http.Error(w, "No link sets found for provided IDs", http.StatusNotFound)
		return
	}

	pdfData, err := h.pdfGen.GenerateReport(linkSets)
	if err != nil {
		log.Printf("Failed to generate PDF: %v", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=report.pdf")
	w.Write(pdfData)
}

// HealthCheck - проверка здоровья сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// StartShutdown инициирует процесс остановки
func (h *Handler) StartShutdown() {
	h.mu.Lock()
	h.shutdown = true
	h.mu.Unlock()
}

// WaitForActiveJobs ожидает завершения активных задач
func (h *Handler) WaitForActiveJobs() {
	h.activeJobs.Wait()
}

// ProcessPendingTasks обрабатывает сохранённые задачи после перезапуска
func (h *Handler) ProcessPendingTasks(ctx context.Context) {
	tasks := h.storage.GetAndClearPendingTasks()
	if len(tasks) == 0 {
		return
	}

	log.Printf("Processing %d pending tasks from previous run", len(tasks))

	for _, task := range tasks {
		results := h.checker.CheckLinks(ctx, task.Links)
		h.storage.AddLinkSet(task.ID, results)
		log.Printf("Completed pending task #%d", task.ID)
	}

	if err := h.storage.Save(); err != nil {
		log.Printf("Failed to save after processing pending tasks: %v", err)
	}
}
