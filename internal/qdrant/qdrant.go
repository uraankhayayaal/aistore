package qdrant

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// Client — обёртка над Qdrant API.
//
// ВАЖНО: это базовая рабочая заглушка. Вместо обращения к реальному серверу Qdrant
// точки хранятся в памяти процесса. Косинусное расстояние считается напрямую.
// Для продакшена замените тело методов на вызовы официального Go SDK
// (github.com/qdrant/go-client) с таким же набором сигнатур.
type Client struct {
	mu          sync.RWMutex
	collections map[string]*collection
}

// collection — in-memory аналог коллекции Qdrant.
type collection struct {
	vectorSize int
	points     []*Point
	nextID     uint64
}

// Point — точка в векторном пространстве с метаданными (payload). Соответствует
// структуре qdrant.PointStruct официального SDK.
type Point struct {
	ID      uint64
	Vector  []float32
	Payload map[string]interface{}
	Score   float32
}

// Chunk представляет текстовый фрагмент с метаданными для хранения в Qdrant.
type Chunk struct {
	Text     string
	Metadata map[string]interface{}
}

// NewClient создаёт клиент векторной базы данных.
func NewClient(host string) (*Client, error) {
	log.Printf("Подключение к Qdrant (in-memory заглушка): %s", host)
	return &Client{
		collections: make(map[string]*collection),
	}, nil
}

// CreateCollection создаёт коллекцию с указанной размерностью вектора.
func (c *Client) CreateCollection(ctx context.Context, collectionName string, vectorSize int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.collections[collectionName]; ok {
		log.Printf("Коллекция %s уже существует", collectionName)
		return nil
	}

	c.collections[collectionName] = &collection{
		vectorSize: vectorSize,
		points:     []*Point{},
		nextID:     1,
	}
	log.Printf("Создана коллекция %s с размерностью вектора %d", collectionName, vectorSize)
	return nil
}

// EnsureCollection создаёт коллекцию, если её нет (идемпотентно).
func (c *Client) EnsureCollection(ctx context.Context, collectionName string, vectorSize int) error {
	return c.CreateCollection(ctx, collectionName, vectorSize)
}

// UpsertChunk сохраняет текстовый фрагмент с embedding-вектором.
func (c *Client) UpsertChunk(ctx context.Context, collectionName string, chunk Chunk, vector []float32) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	col, err := c.getCollection(collectionName)
	if err != nil {
		return err
	}
	if col.vectorSize != len(vector) {
		return fmt.Errorf("размерность вектора %d не совпадает с размерностью коллекции %d", len(vector), col.vectorSize)
	}

	// метаданные должны всегда содержать обязательные поля
	payload := make(map[string]interface{}, len(chunk.Metadata))
	for k, v := range chunk.Metadata {
		payload[k] = v
	}
	payload["text"] = chunk.Text

	col.points = append(col.points, &Point{
		ID:      col.nextID,
		Vector:  vector,
		Payload: payload,
	})
	col.nextID++
	return nil
}

// Search выполняет семантический поиск по косинусной близости и возвращает
// топ-N наиболее похожих точек с метаданными и оценкой совпадения.
func (c *Client) Search(ctx context.Context, collectionName string, vector []float32, limit int32) ([]*Point, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	col, err := c.getCollection(collectionName)
	if err != nil {
		return nil, err
	}

	type scored struct {
		point *Point
		sim   float32
	}
	all := make([]scored, 0, len(col.points))
	for _, p := range col.points {
		sim := cosineSimilarity(vector, p.Vector)
		all = append(all, scored{point: p, sim: sim})
	}

	// сортировка по убыванию схожести (простая сортировка вставками для малых объёмов)
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j-1].sim < all[j].sim; j-- {
			all[j-1], all[j] = all[j], all[j-1]
		}
	}

	n := int(limit)
	if len(all) < n {
		n = len(all)
	}

	results := make([]*Point, 0, n)
	for i := 0; i < n; i++ {
		p := &Point{
			ID:      all[i].point.ID,
			Vector:  all[i].point.Vector,
			Payload: all[i].point.Payload,
			Score:   all[i].sim,
		}
		results = append(results, p)
	}

	return results, nil
}

// GetCollectionStats возвращает статистику коллекции (количество точек и т.д.).
type CollectionStats struct {
	CollectionName string
	PointsCount    int64
	VectorSize     int
}

// GetCollectionStats возвращает количество точек в коллекции и размерность вектора.
func (c *Client) GetCollectionStats(ctx context.Context, collectionName string) (*CollectionStats, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	col, err := c.getCollection(collectionName)
	if err != nil {
		return nil, err
	}

	return &CollectionStats{
		CollectionName: collectionName,
		PointsCount:    int64(len(col.points)),
		VectorSize:     col.vectorSize,
	}, nil
}

// getCollection возвращает коллекцию по имени или ошибку.
func (c *Client) getCollection(name string) (*collection, error) {
	col, ok := c.collections[name]
	if !ok {
		return nil, fmt.Errorf("коллекция %q не найдена", name)
	}
	return col, nil
}

// cosineSimilarity вычисляет косинусную схожесть между двумя векторами (0..1).
func cosineSimilarity(a, b []float32) float32 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

var _ = time.Now // сохраняем импорт time для будущего использования
