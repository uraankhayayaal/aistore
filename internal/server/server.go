package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"knowledge-base/internal/config"
	"knowledge-base/internal/ollama"
	"knowledge-base/internal/qdrant"
	"knowledge-base/internal/sync"
)

// Server — HTTP-сервер, предоставляющий отладочный веб-интерфейс для RAG-системы.
type Server struct {
	config       *config.Config
	qdrantClient *qdrant.Client
	ollamaClient *ollama.Client
	syncManager  *sync.SyncManager
	server       *http.Server
}

// SearchResult — результат поиска: текст фрагмента, метаданные и оценка совпадения.
type SearchResult struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float32                `json:"score"`
}

// SearchResponse — полный ответ на запрос поиска.
type SearchResponse struct {
	Query        string         `json:"query"`
	Response     string         `json:"response"`
	SearchResult []SearchResult `json:"searchResult"`
	LastSync     time.Time      `json:"lastSync"`
	VectorCount  int64          `json:"vectorCount"`
	Error        string         `json:"error,omitempty"`
}

// NewServer создаёт новый экземпляр HTTP-сервера.
func NewServer(cfg *config.Config, qdrantClient *qdrant.Client, ollamaClient *ollama.Client, syncManager *sync.SyncManager) *Server {
	return &Server{
		config:       cfg,
		qdrantClient: qdrantClient,
		ollamaClient: ollamaClient,
		syncManager:  syncManager,
	}
}

// Start запускает HTTP-сервер на порту 8080.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.indexHandler)
	mux.HandleFunc("/search", s.searchHandler)

	s.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("HTTP-сервер запущен на порту :8080")
	return s.server.ListenAndServe()
}

// indexHandler обрабатывает корневой маршрут и отображает HTML-страницу поиска.
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("internal/server/templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats, _ := s.qdrantClient.GetCollectionStats(r.Context(), "rag_collection")
	var vectorCount int64
	lastSync := time.Time{}
	if s.syncManager != nil {
		lastSync = s.syncManager.LastSyncTime()
	}
	if stats != nil {
		vectorCount = stats.PointsCount
	}

	data := map[string]interface{}{
		"LastSync":    lastSync,
		"VectorCount": vectorCount,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Ошибка выполнения шаблона: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// searchHandler обрабатывает POST-запросы поиска:
// embedding запроса → поиск Топ-5 в Qdrant → сбор контекста → генерация ответа.
func (s *Server) searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешён", http.StatusMethodNotAllowed)
		return
	}

	query := r.FormValue("query")
	if query == "" {
		http.Error(w, "Параметр query обязателен", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ctx := r.Context()

	// 1. векторизация запроса через Ollama
	vector, err := s.ollamaClient.Embed(ctx, query, s.config.ModelEmbedding)
	if err != nil {
		s.writeJSON(w, SearchResponse{Query: query, Error: fmt.Sprintf("Ошибка векторизации запроса: %v", err)})
		return
	}

	// 2. поиск Топ-5 похожих векторов в Qdrant
	points, err := s.qdrantClient.Search(ctx, "rag_collection", vector, 5)
	if err != nil {
		s.writeJSON(w, SearchResponse{Query: query, Error: fmt.Sprintf("Ошибка поиска: %v", err)})
		return
	}

	// 3. сбор результатов и построение контекста
	var results []SearchResult
	var contextParts []string
	for _, p := range points {
		res := SearchResult{
			Text:     strVal(p.Payload["text"]),
			Metadata: p.Payload,
			Score:    p.Score,
		}
		results = append(results, res)

		source := strVal(p.Payload["source"])
		title := strVal(p.Payload["title"])
		url := strVal(p.Payload["url"])
		updated := strVal(p.Payload["updated_at"])

		contextParts = append(contextParts, fmt.Sprintf(
			"Источник: %s\nЗаголовок: %s\nURL: %s\nДата обновления: %s\nСодержимое:\n%s",
			source, title, url, updated, res.Text,
		))
	}

	// 4. формирование промпта на русском языке по контексту
	prompt := fmt.Sprintf(`Ты — интеллектуальный помощник корпоративной базы знаний.
Отвечай строго на русском языке, используя только предоставленный контекст.
Если в контексте нет ответа, так и скажи. В конце ответа укажи ссылки на источники.

Контекст:
%s

Вопрос пользователя: %s

Ответ:`, joinContext(contextParts), query)

	// 5. генерация финального ответа через Ollama
	response, err := s.ollamaClient.Generate(ctx, prompt, s.config.ModelGeneration)
	if err != nil {
		s.writeJSON(w, SearchResponse{Query: query, Error: fmt.Sprintf("Ошибка генерации: %v", err)})
		return
	}

	stats, _ := s.qdrantClient.GetCollectionStats(ctx, "rag_collection")
	var vectorCount int64
	lastSync := time.Time{}
	if s.syncManager != nil {
		lastSync = s.syncManager.LastSyncTime()
	}
	if stats != nil {
		vectorCount = stats.PointsCount
	}

	s.writeJSON(w, SearchResponse{
		Query:        query,
		Response:     response,
		SearchResult: results,
		LastSync:     lastSync,
		VectorCount:  vectorCount,
	})
}

// writeJSON пишет структуру в JSON-ответ.
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
	}
}

// strVal безопасно получает строковое значение из метаданных.
func strVal(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// joinContext объединяет части контекста в одну строку.
func joinContext(parts []string) string {
	out := ""
	for i, p := range parts {
		out += fmt.Sprintf("--- Фрагмент %d ---\n%s\n\n", i+1, p)
	}
	return out
}
