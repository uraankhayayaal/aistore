package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config содержит всю конфигурацию приложения, загружаемую из переменных окружения.
type Config struct {
	OllamaHost      string        // адрес Ollama API
	ModelEmbedding  string        // модель для генерации embeddings
	ModelGeneration string        // модель для генерации текста
	QdrantHost      string        // адрес Qdrant (gRPC)
	EvaURL          string        // URL API EvaProject
	EvaToken        string        // токен доступа к EvaProject
	GitlabURL       string        // URL GitLab-инстанса
	GitlabToken     string        // токен доступа к GitLab
	SyncInterval    time.Duration // интервал автоматической синхронизации
}

// LoadConfig загружает конфигурацию из переменных окружения (.env файл).
func LoadConfig() (*Config, error) {
	cfg := &Config{
		OllamaHost:      getEnv("OLLAMA_HOST", "http://localhost:11434"),
		ModelEmbedding:  getEnv("MODEL_EMBEDDING", "nomic-embed-text"),
		ModelGeneration: getEnv("MODEL_GENERATION", "qwen2.5:7b"),
		QdrantHost:      getEnv("QDRANT_HOST", "localhost:6334"),
		EvaURL:          getEnv("EVA_URL", ""),
		EvaToken:        getEnv("EVA_TOKEN", ""),
		GitlabURL:       getEnv("GITLAB_URL", ""),
		GitlabToken:     getEnv("GITLAB_TOKEN", ""),
	}

	// парсинг интервала синхронизации (по умолчанию 10 минут)
	intervalStr := getEnv("SYNC_INTERVAL", "10m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Printf("Некорректный SYNC_INTERVAL: %s, используется значение по умолчанию 10m", intervalStr)
		interval = 10 * time.Minute
	}
	cfg.SyncInterval = interval

	return cfg, nil
}

// getEnv возвращает значение переменной окружения или значение по умолчанию.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt возвращает целочисленное значение переменной окружения или значение по умолчанию.
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}
