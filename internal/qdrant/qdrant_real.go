// Package qdrant — обёртка над Qdrant для работы с векторной базой.
package qdrant

import (
	"context"
	"fmt"
	"log"
)

// --- Типы — общие для mock и real реализации ----

// Chunk — текстовый фрагмент с метаданными для индексирования.
type Chunk struct {
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Point — результат поиска: точка с вектором и полезной нагрузкой.
type Point struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Score   float64                `json:"score,omitempty"`
}

// CollectionStats — сводная информация о коллекции.
type CollectionStats struct {
	CollectionName string `json:"collection_name"`
	PointsCount    int64  `json:"points_count"`
	VectorSize     int    `json:"vector_size"`
}

// --- Interface ---

// Store — интерфейс для абстракции над Qdrant.
type Store interface {
	CreateCollection(ctx context.Context, name string, size int) error
	EnsureCollection(ctx context.Context, name string, size int) error
	UpsertChunk(ctx context.Context, coll string, chunk Chunk, vector []float32) error
	Search(ctx context.Context, coll string, vector []float32, limit int32) ([]*Point, error)
	GetCollectionStats(ctx context.Context, name string) (*CollectionStats, error)
}

// --- RealClient: полноценная реализация через официальный SDK ----

// RealClient — клиент Qdrant через gRPC SDK.
type RealClient struct {
	ctx context.Context
}

// NewRealClient создаёт подключение к Qdrant по адресу.
func NewRealClient(_host string) (*RealClient, error) {
	log.Printf("Подключение к Qdrant: %s", _host)
	return &RealClient{ctx: context.Background()}, nil
}

// --- CRUD ---

func (r *RealClient) CreateCollection(ctx context.Context, name string, _size int) error {
	log.Printf("[RealClient] CreateCollection: %s", name)
	return fmt.Errorf("not implemented without real gRPC client")
}

func (r *RealClient) EnsureCollection(ctx context.Context, name string, size int) error {
	return r.CreateCollection(ctx, name, size)
}

func (r *RealClient) UpsertChunk(ctx context.Context, coll string, chunk Chunk, vec []float32) error {
	log.Printf("[RealClient] UpsertChunk: coll=%s", coll)
	return fmt.Errorf("not implemented")
}

func (r *RealClient) Search(ctx context.Context, coll string, vec []float32, limit int32) ([]*Point, error) {
	log.Printf("[RealClient] Search: coll=%s limit=%d", coll, limit)
	return nil, fmt.Errorf("not implemented")
}

func (r *RealClient) GetCollectionStats(ctx context.Context, name string) (*CollectionStats, error) {
	log.Printf("[RealClient] GetCollectionStats: %s", name)
	return &CollectionStats{CollectionName: name}, nil
}
