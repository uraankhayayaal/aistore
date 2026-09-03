package qdrant

import (
	"context"
	"fmt"
	"log"
)

// Client — обёртка над Qdrant API.
type Client struct {
	realClient *RealClient
}

// NewClient создаёт клиент векторной базы данных.
func NewClient(host string) (*Client, error) {
	log.Printf("Подключение к реальному Qdrant: %s", host)

	realClient, err := NewRealClient(host)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания реального клиента Qdrant: %w", err)
	}

	return &Client{
		realClient: realClient,
	}, nil
}

// CreateCollection создаёт коллекцию с указанной размерностью вектора.
func (c *Client) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return c.realClient.CreateCollection(ctx, collectionName, vectorSize)
}

// EnsureCollection создаёт коллекцию, если её нет (идемпотентно).
func (c *Client) EnsureCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return c.realClient.EnsureCollection(ctx, collectionName, vectorSize)
}

// UpsertChunk сохраняет текстовый фрагмент с embedding-вектором.
func (c *Client) UpsertChunk(ctx context.Context, collectionName string, chunk Chunk, vector []float32) error {
	return c.realClient.UpsertChunk(ctx, collectionName, chunk, vector)
}

// Search выполняет семантический поиск по косинусной близости и возвращает
// топ-N наиболее похожих точек с метаданными и оценкой совпадения.
func (c *Client) Search(ctx context.Context, collectionName string, vector []float32, limit int32) ([]*Point, error) {
	return c.realClient.Search(ctx, collectionName, vector, limit)
}

// GetCollectionStats возвращает статистику коллекции (количество точек и т.д.).
func (c *Client) GetCollectionStats(ctx context.Context, collectionName string) (*CollectionStats, error) {
	return c.realClient.GetCollectionStats(ctx, collectionName)
}
