package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	qdrantpb "github.com/qdrant/go-client/qdrant"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RealClient — реализация клиента Qdrant. Хранит gRPC-соединение и points-клиент.
type RealClient struct {
	conn          *grpc.ClientConn
	pointsCl      qdrantpb.PointsClient
	collectionsCl qdrantpb.CollectionsClient
	host          string
}

// NewRealClient создаёт RealClient, подключаясь к Qdrant по адресу.
func NewRealClient(host string) (*RealClient, error) {
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("подключение к qdrant %s: %w", host, err)
	}
	return &RealClient{
		conn:          conn,
		pointsCl:      qdrantpb.NewPointsClient(conn),
		collectionsCl: qdrantpb.NewCollectionsClient(conn),
		host:          host,
	}, nil
}

// Close закрывает gRPC-соединение.
func (c *RealClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// --- Qdrant operations ---

// CreateCollection создаёт коллекцию.
func (c *RealClient) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	var defaultSegmentNumber uint64 = 2
	_, err := c.collectionsCl.Create(ctx, &qdrantpb.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrantpb.VectorsConfig{
			Config: &qdrantpb.VectorsConfig_Params{
				Params: &qdrantpb.VectorParams{
					Size:     uint64(vectorSize),
					Distance: qdrantpb.Distance_Dot,
				},
			},
		},
		OptimizersConfig: &qdrantpb.OptimizersConfigDiff{
			DefaultSegmentNumber: &defaultSegmentNumber,
		},
	})
	return err
}

// EnsureCollection создаёт коллекцию, если её нет (идемпотентно).
func (c *RealClient) EnsureCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return c.CreateCollection(ctx, collectionName, vectorSize)
}

// UpsertChunk сохраняет текстовый фрагмент с embedding-вектором.
func (c *RealClient) UpsertChunk(ctx context.Context, collectionName string, chunk Chunk, vector []float32) error {
	newID := generatePointID()

	numID := 0
	if v, ok := chunk.Metadata["id"]; ok {
		switch r := v.(type) {
		case float64:
			numID = int(r)
		case int:
			numID = r
		}
	}

	now := time.Now()
	fmt.Printf("[%s] UpsertChunk: collection=%s, chunkID=%d, pointID=%s\n",
		now.Format(time.RFC3339), collectionName, numID, newID)

	payload := toQdrantPayload(map[string]any{
		"id":           numID,
		"kind":         chunk.Metadata["kind"],
		"body":         chunk.Text,
		"start_index":  chunk.Metadata["start_index"],
		"chunk_number": chunk.Metadata["chunk_number"],
	})

	upsertPoints := &qdrantpb.UpsertPoints{
		CollectionName: collectionName,
		Wait:           ptrBool(true),
		Points: []*qdrantpb.PointStruct{
			{
				Id:      pointID(newID),
				Payload: payload,
				Vectors: &qdrantpb.Vectors{
					VectorsOptions: &qdrantpb.Vectors_Vector{
						Vector: &qdrantpb.Vector{Data: vector},
					},
				},
			},
		},
	}

	_, err := c.pointsCl.Upsert(ctx, upsertPoints)
	if err != nil {
		return fmt.Errorf("upsert chunk: %w", err)
	}
	fmt.Printf("[%s] UpsertChunk: OK, chunkID=%d, pointID=%s\n", time.Now().Format(time.RFC3339), numID, newID)
	return nil
}

// Search выполняет семантический поиск.
func (c *RealClient) Search(ctx context.Context, collectionName string, vector []float32, limit int32) ([]*Point, error) {
	// TODO: реализовать поиск.
	return nil, nil
}

// GetCollectionStats возвращает статистику коллекции.
func (c *RealClient) GetCollectionStats(ctx context.Context, collectionName string) (*CollectionStats, error) {
	// TODO: реализовать.
	return nil, nil
}

// HealthCheck проверяет доступность Qdrant.
func (c *RealClient) HealthCheck(ctx context.Context) error {
	fmt.Println("Health check called (stub).")
	return nil
}

// generatePointID возвращает "k-b-c-" + UUID v4.
func generatePointID() string {
	return fmt.Sprintf("k-b-c-%s", uuid.New().String())
}

// --- --- Подмопомощь---

// sanitizeBody заменяет переносы строк пробелами.
func sanitizeBody(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "  ", " ")
	return strings.TrimSpace(s)
}

// jsonToChunk parse JSON to Chunks.
func jsonToChunk(b []byte) ([]Chunk, error) {
	var chunks []Chunk
	if err := json.Unmarshal(b, &chunks); err != nil {
		return nil, fmt.Errorf("parse chunks json: %w", err)
	}
	return chunks, nil
}

// safeFloat64
func safeFloat64(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// safeInt
func safeInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

func extractValue(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func ptrBool(b bool) *bool {
	return &b
}

func pointID(uuidStr string) *qdrantpb.PointId {
	return &qdrantpb.PointId{
		PointIdOptions: &qdrantpb.PointId_Uuid{Uuid: uuidStr},
	}
}

func toQdrantPayload(src map[string]any) map[string]*qdrantpb.Value {
	dst := make(map[string]*qdrantpb.Value, len(src))
	for k, v := range src {
		dst[k] = anyToValue(v)
	}
	return dst
}

func anyToValue(v any) *qdrantpb.Value {
	switch v := v.(type) {
	case string:
		return &qdrantpb.Value{Kind: &qdrantpb.Value_StringValue{StringValue: v}}
	case int:
		return &qdrantpb.Value{Kind: &qdrantpb.Value_IntegerValue{IntegerValue: int64(v)}}
	case int64:
		return &qdrantpb.Value{Kind: &qdrantpb.Value_IntegerValue{IntegerValue: v}}
	case float64:
		return &qdrantpb.Value{Kind: &qdrantpb.Value_DoubleValue{DoubleValue: v}}
	case bool:
		return &qdrantpb.Value{Kind: &qdrantpb.Value_BoolValue{BoolValue: v}}
	case nil:
		return &qdrantpb.Value{}
	default:
		return &qdrantpb.Value{Kind: &qdrantpb.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
}
