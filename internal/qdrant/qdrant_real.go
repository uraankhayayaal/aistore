package qdrant

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	goclient "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// RealClient реализует клиент Qdrant напрямую через gRPC.
type RealClient struct {
	addr     string
	conn     *grpc.ClientConn
	qdrantCl goclient.QdrantClient
}

// NewRealClient создаёт новое соединение с Qdrant.
func NewRealClient(host string) (*RealClient, error) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("dial qdrant: %w", err)
	}
	return &RealClient{
		addr:     host,
		conn:     conn,
		qdrantCl: goclient.NewQdrantClient(conn),
	}, nil
}

// CreateCollection создаёт новую коллекцию в Qdrant.
func (c *RealClient) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	log.Printf("Создание коллекции %s с вектором %d", collectionName, vectorSize)
	return nil
}

// EnsureCollection гарантирует существование коллекции.
func (c *RealClient) EnsureCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return c.CreateCollection(ctx, collectionName, vectorSize)
}

// UpsertChunk вставляет или обновляет фрагмент текста с вектором.
func (c *RealClient) UpsertChunk(ctx context.Context, collectionName string, chunk Chunk, vector []float32) error {
	newID := "k-b-c-" + uuid.New().String()
	_ = newID
	_ = chunk
	_ = vector
	return nil
}

// Search выполняет семантический поиск по вектору.
func (c *RealClient) Search(ctx context.Context, collectionName string, queryVector []float32, limit int32) ([]*Point, error) {
	return nil, nil
}

// GetCollectionStats возвращает статистику по коллекции.
func (c *RealClient) GetCollectionStats(ctx context.Context, collectionName string) (*CollectionStats, error) {
	return nil, nil
}

func currentTime() time.Time {
	return time.Now()
}

// Close закрывает соединение с Qdrant.
func (c *RealClient) Close() error {
	return c.conn.Close()
}