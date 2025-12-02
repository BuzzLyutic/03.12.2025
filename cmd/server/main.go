package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/BuzzLyutic/03.12.2025/internal/config"
	"github.com/BuzzLyutic/03.12.2025/internal/handler"
	"github.com/BuzzLyutic/03.12.2025/internal/pdf"
	"github.com/BuzzLyutic/03.12.2025/internal/service"
	"github.com/BuzzLyutic/03.12.2025/internal/storage"
)

func main() {
	// Загрузка конфигурации
	cfg := config.Load()

	// Инициализация хранилища
	store, err := storage.New(cfg.DataPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Инициализация сервисов
	checker := service.NewChecker(cfg.CheckTimeout)
	pdfGen := pdf.NewGenerator()
	h := handler.New(checker, store, pdfGen)

	// Обработка сохранённых задач после перезапуска
	h.ProcessPendingTasks(context.Background())

	// Настройка маршрутов
	mux := http.NewServeMux()
	mux.HandleFunc("/check", h.CheckLinks)
	mux.HandleFunc("/report", h.GetReport)
	mux.HandleFunc("/health", h.HealthCheck)

	server := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Канал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	go func() {
		log.Printf("Server starting on %s", cfg.ServerAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Ожидание сигнала
	<-quit
	log.Println("Shutdown signal received...")

	// Новые задачи будут сохраняться как pending
	h.StartShutdown()

	// Создание контекста с таймаутом для shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Ожидание завершения активных задач
	log.Println("Waiting for active jobs to complete...")
	done := make(chan struct{})
	go func() {
		h.WaitForActiveJobs()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All active jobs completed")
	case <-ctx.Done():
		log.Println("Timeout waiting for active jobs")
	}

	// Сохранение состояния
	if err := store.Save(); err != nil {
		log.Printf("Failed to save storage: %v", err)
	}

	// Остановка HTTP сервера
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped gracefully")
}
