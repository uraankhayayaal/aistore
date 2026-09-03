package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"knowledge-base/internal/config"
	"knowledge-base/internal/ollama"
	"knowledge-base/internal/qdrant"
	"knowledge-base/internal/server"
	"knowledge-base/internal/sync"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// загрузка конфигурации из .env
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// инициализация клиента Qdrant
	qdrantClient, err := qdrant.NewClient(cfg.QdrantHost)
	if err != nil {
		log.Fatalf("Ошибка инициализации клиента Qdrant: %v", err)
	}

	// инициализация клиента Ollama
	ollamaClient, err := ollama.NewClient(cfg.OllamaHost)
	if err != nil {
		log.Fatalf("Ошибка инициализации клиента Ollama: %v", err)
	}

	// запуск фоновой синхронизации данных
	syncManager := sync.NewSyncManager(cfg, qdrantClient, ollamaClient)
	go func() {
		if err := syncManager.Start(ctx); err != nil {
			log.Printf("Ошибка синхронизации: %v", err)
		}
	}()

	// запуск HTTP-сервера
	srv := server.NewServer(cfg, qdrantClient, ollamaClient, syncManager)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("Ошибка сервера: %v", err)
		}
	}()

	// ожидание сигнала завершения (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Завершение работы...")
	cancel()
	time.Sleep(2 * time.Second)
}
