package qdrant

// Point — точка в векторном пространстве с метаданными (payload). Соответствует
// структуре qdrant.PointStruct официального SDK.
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
	Score   float32
}

// Chunk представляет текстовый фрагмент с метаданными для хранения в Qdrant.
type Chunk struct {
	Text     string
	Metadata map[string]interface{}
}

// CollectionStats — статистика коллекции Qdrant.
type CollectionStats struct {
	CollectionName string
	PointsCount    int64
	VectorSize     int
}
