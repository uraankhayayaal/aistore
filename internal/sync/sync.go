package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"knowledge-base/internal/config"
	"knowledge-base/internal/ollama"
	"knowledge-base/internal/qdrant"
)

// SyncManager управляет синхронизацией данных из внешних источников (EvaProject, GitLab)
// с векторной базой данных Qdrant.
type SyncManager struct {
	config       *config.Config
	qdrantClient *qdrant.Client
	ollamaClient *ollama.Client
	evaClient    *EvaClient
	gitlabClient *GitLabClient
	lastSyncTime time.Time
	syncDone     chan bool
}

// SyncState хранит состояние последней синхронизации для инкрементальных обновлений.
type SyncState struct {
	LastSync time.Time `json:"last_sync"`
}

// NewSyncManager создаёт новый менеджер синхронизации.
func NewSyncManager(cfg *config.Config, qdrantClient *qdrant.Client, ollamaClient *ollama.Client) *SyncManager {
	return &SyncManager{
		config:       cfg,
		qdrantClient: qdrantClient,
		ollamaClient: ollamaClient,
		evaClient:    NewEvaClient(cfg.EvaURL, cfg.EvaToken),
		gitlabClient: NewGitLabClient(cfg.GitlabURL, cfg.GitlabToken),
		syncDone:     make(chan bool),
	}
}

// LastSyncTime возвращает время последней завершённой синхронизации.
func (sm *SyncManager) LastSyncTime() time.Time {
	return sm.lastSyncTime
}

// Start запускает цикл синхронизации: холодный старт (если нужно) + периодический тикер.
func (sm *SyncManager) Start(ctx context.Context) error {
	// определение размерности вектора через тестовый embedding
	vectorSize, err := sm.getVectorSize()
	if err != nil {
		return fmt.Errorf("не удалось определить размерность вектора: %w", err)
	}

	// создание коллекции в Qdrant (если не существует)
	err = sm.qdrantClient.CreateCollection(ctx, "rag_collection", vectorSize)
	if err != nil {
		return fmt.Errorf("не удалось создать коллекцию: %w", err)
	}

	// загрузка времени последней синхронизации
	sm.loadLastSync()

	// холодный старт: полный обход всех данных, если синхронизация ещё не проводилась
	if sm.lastSyncTime.IsZero() {
		log.Println("Выполнение холодного старта (полная синхронизация)...")
		err := sm.performFullSync(ctx)
		if err != nil {
			return fmt.Errorf("ошибка холодного старта: %w", err)
		}
		sm.lastSyncTime = time.Now()
	}

	// запуск периодической синхронизации по таймеру
	ticker := time.NewTicker(sm.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Менеджер синхронизации остановлен")
			return ctx.Err()
		case <-ticker.C:
			log.Println("Выполнение запланированной синхронизации...")
			err := sm.performIncrementalSync(ctx)
			if err != nil {
				log.Printf("Ошибка синхронизации: %v", err)
			}
			sm.lastSyncTime = time.Now()
			sm.saveLastSync()
		}
	}
}

// loadLastSync загружает время последней синхронизации из локального JSON-файла.
func (sm *SyncManager) loadLastSync() {
	stateFile := filepath.Join(".", ".sync_state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		// файл не существует — это первый запуск
		return
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("Ошибка чтения состояния синхронизации: %v", err)
		return
	}

	sm.lastSyncTime = state.LastSync
}

// saveLastSync сохраняет текущее время как время последней синхронизации.
func (sm *SyncManager) saveLastSync() {
	state := SyncState{
		LastSync: sm.lastSyncTime,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		log.Printf("Ошибка сериализации состояния синхронизации: %v", err)
		return
	}

	stateFile := filepath.Join(".", ".sync_state.json")
	err = os.WriteFile(stateFile, data, 0644)
	if err != nil {
		log.Printf("Ошибка записи состояния синхронизации: %v", err)
	}
}

// getVectorSize определяет размерность embedding-вектора модели через тестовый запрос.
func (sm *SyncManager) getVectorSize() (int, error) {
	ctx := context.Background()
	vector, err := sm.ollamaClient.Embed(ctx, "test", sm.config.ModelEmbedding)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить размерность вектора: %w", err)
	}
	return len(vector), nil
}

// ContentItem — единица контента из внешнего источника.
type ContentItem struct {
	Source    string // "gitlab" | "evaproject"
	Title     string
	URL       string
	Text      string
	UpdatedAt string
}

// performFullSync выполняет полную синхронизацию всех данных из внешних источников.
func (sm *SyncManager) performFullSync(ctx context.Context) error {
	log.Println("Холодный старт: полный обход всех источников данных...")
	items := sm.fetchAllContent(ctx)
	return sm.processItems(ctx, items)
}

// performIncrementalSync выполняет инкрементальную синхронизацию,
// запрашивая только изменения после последнего обновления.
func (sm *SyncManager) performIncrementalSync(ctx context.Context) error {
	log.Printf("Инкрементальная синхронизация (изменения после %s)...", sm.lastSyncTime.Format(time.RFC3339))
	items := sm.fetchChangedContent(ctx, sm.lastSyncTime)
	return sm.processItems(ctx, items)
}

// fetchAllContent получает все данные из источников (EvaProject + GitLab).
// Если источник не настроен (URL пуст), используется демо-контент для источника.
func (sm *SyncManager) fetchAllContent(ctx context.Context) []ContentItem {
	items := make([]ContentItem, 0)

	// EvaProject
	if sm.evaClient.Enabled() {
		eva, err := sm.evaClient.FetchAll(ctx)
		if err != nil {
			log.Printf("EvaProject: ошибка загрузки всех задач: %v", err)
		} else {
			items = append(items, eva...)
			log.Printf("EvaProject: получено задач: %d", len(eva))
		}
	} else {
		log.Println("EvaProject не настроен, используется демо-контент")
		items = append(items, sm.demoContent(SourceEvaProject)...)
	}

	// GitLab
	if sm.gitlabClient.Enabled() {
		gl, err := sm.gitlabClient.FetchAll(ctx)
		if err != nil {
			log.Printf("GitLab: ошибка загрузки проектов: %v", err)
		} else {
			items = append(items, gl...)
			log.Printf("GitLab: получено элементов: %d", len(gl))
		}
	} else {
		log.Println("GitLab не настроен, используется демо-контент")
		items = append(items, sm.demoContent(SourceGitLab)...)
	}

	return items
}

// fetchChangedContent получает только изменённые данные после заданного времени.
// Если источник не настроен, возвращает демо-контент.
func (sm *SyncManager) fetchChangedContent(ctx context.Context, since time.Time) []ContentItem {
	items := make([]ContentItem, 0)

	if sm.evaClient.Enabled() {
		eva, err := sm.evaClient.FetchSince(ctx, since)
		if err != nil {
			log.Printf("EvaProject: ошибка загрузки изменений: %v", err)
		} else {
			items = append(items, eva...)
		}
	}

	if sm.gitlabClient.Enabled() {
		gl, err := sm.gitlabClient.FetchSince(ctx, since)
		if err != nil {
			log.Printf("GitLab: ошибка загрузки изменений: %v", err)
		} else {
			items = append(items, gl...)
		}
	}

	// если источники не настроены — демо-контент для проверки
	if !sm.evaClient.Enabled() && !sm.gitlabClient.Enabled() {
		items = append(items, sm.demoContent(SourceEvaProject)...)
		items = append(items, sm.demoContent(SourceGitLab)...)
	}

	return items
}

// processItems разбивает контент на чанки, создаёт embeddings и сохраняет в Qdrant.
func (sm *SyncManager) processItems(ctx context.Context, items []ContentItem) error {
	for _, item := range items {
		chunks := sm.chunkText(item.Text, 1500, 200)

		for _, chunk := range chunks {
			vector, err := sm.ollamaClient.Embed(ctx, chunk.Text, sm.config.ModelEmbedding)
			if err != nil {
				log.Printf("Ошибка создания embedding: %v", err)
				continue
			}

			metadata := map[string]interface{}{
				"source":     item.Source,
				"url":        item.URL,
				"title":      item.Title,
				"text":       chunk.Text,
				"updated_at": item.UpdatedAt,
			}

			err = sm.qdrantClient.UpsertChunk(ctx, "rag_collection", qdrant.Chunk{
				Text:     chunk.Text,
				Metadata: metadata,
			}, vector)
			if err != nil {
				log.Printf("Ошибка сохранения фрагмента: %v", err)
				continue
			}
		}
	}

	log.Printf("Обработано элементов: %d", len(items))
	return nil
}

// demoContent возвращает демонстрационные данные для указанного источника,
// используемые, когда реальный источник не настроен.
func (sm *SyncManager) demoContent(source string) []ContentItem {
	now := time.Now().Format(time.RFC3339)

	if source == SourceGitLab {
		return []ContentItem{
			{
				Source:    SourceGitLab,
				Title:     "Wiki: Clean Architecture",
				URL:       "https://gitlab.example/project/wiki/clean-architecture",
				Text:      "Описание принципов чистой архитектуры: разделение на слои, инверсия зависимостей, изоляция бизнес-логики.",
				UpdatedAt: now,
			},
			{
				Source:    SourceGitLab,
				Title:     "Репозиторий: qdrant-go-sdk",
				URL:       "https://gitlab.example/qdrant/go-client",
				Text:      "Примеры использования официального Go SDK для Qdrant: создание коллекции, upsert точек, семантический поиск.",
				UpdatedAt: now,
			},
		}
	}

	return []ContentItem{
		{
			Source:    SourceEvaProject,
			Title:     "Задача: Архитектура RAG-системы",
			URL:       "https://evaproject.example/tasks/101",
			Text:      "Проектирование RAG-системы на Go: выбор стека, модульная структура, clean architecture, интеграция с Ollama и Qdrant.",
			UpdatedAt: now,
		},
		{
			Source:    SourceEvaProject,
			Title:     "Задача: Настройка Ollama",
			URL:       "https://evaproject.example/tasks/102",
			Text:      "Настройка локального Ollama для embeddings и генерации. Загрузка моделей nomic-embed-text и qwen2.5:7b.",
			UpdatedAt: now,
		},
	}
}

// chunkText разбивает текст на фрагменты указанного размера с перекрытием (overlap).
func (sm *SyncManager) chunkText(text string, chunkSize, overlap int) []qdrant.Chunk {
	if len(text) <= chunkSize {
		return []qdrant.Chunk{{Text: text}}
	}

	var chunks []qdrant.Chunk
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunk := text[start:end]
		chunks = append(chunks, qdrant.Chunk{Text: chunk})

		start += chunkSize - overlap
	}

	return chunks
}
