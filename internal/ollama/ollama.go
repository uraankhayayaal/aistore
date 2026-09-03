package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client — HTTP-клиент для взаимодействия с Ollama API.
type Client struct {
	baseURL string
	client  *http.Client
}

// EmbeddingRequest — тело запроса на получение embedding-вектора.
type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// EmbeddingResponse — ответ Ollama API с массивом embeddings.
type EmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// GenerateRequest — тело запроса на генерацию текста.
type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream,omitempty"`
}

// GenerateResponse — ответ Ollama API с сгенерированным текстом.
type GenerateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NewClient создаёт новый HTTP-клиент для Ollama.
func NewClient(baseURL string) (*Client, error) {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Embed отправляет текст в Ollama и возвращает embedding-вектор.
func (c *Client) Embed(ctx context.Context, text string, model string) ([]float32, error) {
	reqBody := EmbeddingRequest{
		Model: model,
		Input: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса embedding: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания HTTP-запроса embedding: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса embedding: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API embedding вернул код статуса %d", resp.StatusCode)
	}

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("ошибка десериализации ответа embedding: %w", err)
	}

	if len(embeddingResp.Embeddings) == 0 {
		return nil, fmt.Errorf("API не вернул ни одного embedding-вектора")
	}

	return embeddingResp.Embeddings[0], nil
}

// Generate отправляет промпт в Ollama и возвращает сгенерированный текст.
func (c *Client) Generate(ctx context.Context, prompt string, model string) (string, error) {
	reqBody := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации запроса генерации: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("ошибка создания HTTP-запроса генерации: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки запроса генерации: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API генерации вернул код статуса %d", resp.StatusCode)
	}

	var generateResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResp); err != nil {
		return "", fmt.Errorf("ошибка десериализации ответа генерации: %w", err)
	}

	return generateResp.Response, nil
}
