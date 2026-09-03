package qdrant

import (
	"context"
	"fmt"
	"log"
	"time"

	qdrant "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RealClient — клиент векторной базы данных Qdrant через официальный SDK.
type RealClient struct {
	client qdrant.QdrantClient
	ctx    context.Context
}

// NewClient создаёт подключение к серверу Qdrant через gRPC.
func NewClient(host string) (*RealClient, error) {
	log.Printf("Подключение к реальному Qdrant: %s", host)

	// Подключаемся по gRPC
	conn, err := grpc.Dial(host, insecure.NewCredentials())
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к Qdrant: %w", err)
	}

	client := qdrant.NewQdrantClient(conn)
	ctx := context.Background()

	return &RealClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// CreateCollection создаёт коллекцию с указанной размерностью.
func (c *RealClient) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	_, err := c.client.CreateCollection(c.ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(vectorSize),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ошибка создания коллекции %s: %w", collectionName, err)
	}
	log.Printf("Создана коллекция %s с размерностью вектора %d", collectionName, vectorSize)
	return nil
}

// EnsureCollection создаёт коллекцию, если она не существует (идемпотентно).
func (c *RealClient) EnsureCollection(ctx context.Context, collectionName string, vectorSize int) error {
	// Просто вызываем CreateCollection — Qdrant сам отвечает за idempotency
	return c.CreateCollection(ctx, collectionName, vectorSize)
}

// UpsertChunk сохраняет текстовый фрагмент с embedding-вектором.
func (c *RealClient) UpsertChunk(ctx context.Context, collectionName string, chunk Chunk, vector []float32) error {
	// Преобразуем metadанные в map[string]interface{} для Qdrant Value
	payload := make(map[string]*qdrant.Value)
	for k, v := range chunk.Metadata {
		payload[k] = valueToQdrantValue(v)
	}
	payload["text"] = &qdrant.Value{Value: &qdrant.Value_TextValue{TextValue: chunk.Text}}

	pointID := &qdrant.PointId{
		Id: &qdrant.PointId_Uuid{Uuid: fmt.Sprintf("%d", time.Now().UnixNano())},
	}

	_, err := c.client.Upsert(c.ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id:      pointID,
				Vectors: &qdrant.Vectors{Vectors: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: vector}}},
				Payload: payload,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ошибка сохранения точки в %s: %w", collectionName, err)
	}
	return nil
}

// Search выполняет семантический поиск по косинусной близости и возвращает топ-N точек.
func (c *RealClient) Search(ctx context.Context, collectionName string, vector []float32, limit int32) ([]*Point, error) {
	resp, err := c.client.Search(c.ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          uint64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска в %s: %w", collectionName, err)
	}

	result := make([]*Point, 0, len(resp.Result))
	for _, scored := range resp.Result {
		payload := make(map[string]interface{})
		for k, v := range scored.Payload {
			payload[k] = qdrantValueToGo(v)
		}

		p := &Point{
			ID:      scored.Id.GetUuid(),
			Vector:  scored.Vector.Data,
			Payload: payload,
			Score:   scored.Score,
		}
		result = append(result, p)
	}

	return result, nil
}

// GetCollectionStats получает информацию о коллекции.
func (c *RealClient) GetCollectionStats(ctx context.Context, collectionName string) (*CollectionStats, error) {
	resp, err := c.client.GetCollectionInfo(c.ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о коллекции %s: %w", collectionName, err)
	}

	stats := &CollectionStats{
		CollectionName: collectionName,
		PointsCount:    resp.PointsCount,
		VectorSize:     int(resp.VectorsConfig.GetParams().Size),
	}
	return stats, nil
}

// valueToQdrantValue преобразует Go-значение в qdrant.Value.
func valueToQdrantValue(v interface{}) *qdrant.Value {
	switch val := v.(type) {
	case string:
		return &qdrant.Value{Value: &qdrant.Value_TextValue{TextValue: val}}
	case int:
		return &qdrant.Value{Value: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
	case float64:
		return &qdrant.Value{Value: &qdrant.Value_DoubleValue{DoubleValue: val}}
	case bool:
		return &qdrant.Value{Value: &qdrant.Value_BoolValue{BoolValue: val}}
	default:
		return &qdrant.Value{Value: &qdrant.Value_TextValue{TextValue: fmt.Sprintf("%v", val)}}
	}
}

// qdrantValueToGo преобразует qdrant.Value в Go-значение.
func qdrantValueToGo(v *qdrant.Value) interface{} {
	switch val := v.GetValue().(type) {
	case *qdrant.Value_TextValue:
		return val.TextValue
	case *qdrant.Value_IntegerValue:
		return val.IntegerValue
	case *qdrant.Value_DoubleValue:
		return val.DoubleValue
	case *qdrant.Value_BoolValue:
		return val.BoolValue
	default:
		return fmt.Sprintf("%v", v)
	}
}
