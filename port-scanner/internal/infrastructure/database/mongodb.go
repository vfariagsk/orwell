package database

import (
	"context"
	"fmt"
	"time"

	"port-scanner/internal/domain"
	"port-scanner/pkg/log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// MongoDBManager manages MongoDB operations for scan results
type MongoDBManager struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// ScanResultDocument represents the MongoDB document structure for scan results
type ScanResultDocument struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	IP            string                 `bson:"ip" json:"ip"`
	IsUp          bool                   `bson:"is_up" json:"is_up"`
	PingTime      time.Duration          `bson:"ping_time" json:"ping_time"`
	ScanStartTime time.Time              `bson:"scan_start_time" json:"scan_start_time"`
	ScanEndTime   time.Time              `bson:"scan_end_time" json:"scan_end_time"`
	Status        string                 `bson:"status" json:"status"`
	Error         string                 `bson:"error,omitempty" json:"error,omitempty"`
	BatchID       string                 `bson:"batch_id" json:"batch_id"`
	WorkerID      string                 `bson:"worker_id" json:"worker_id"`
	Ports         []PortDocument         `bson:"ports" json:"ports"`
	OpenPorts     int                    `bson:"open_ports" json:"open_ports"`
	TotalPorts    int                    `bson:"total_ports" json:"total_ports"`
	ScanDuration  time.Duration          `bson:"scan_duration" json:"scan_duration"`
	CreatedAt     time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
	Metadata      map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// PortDocument represents the MongoDB document structure for ports
type PortDocument struct {
	Number       int                    `bson:"number" json:"number"`
	Status       string                 `bson:"status" json:"status"`
	Service      string                 `bson:"service" json:"service"`
	Banner       string                 `bson:"banner,omitempty" json:"banner,omitempty"`
	Version      string                 `bson:"version,omitempty" json:"version,omitempty"`
	ScanTime     time.Time              `bson:"scan_time" json:"scan_time"`
	ResponseTime time.Duration          `bson:"response_time" json:"response_time"`
	BannerInfo   *BannerInfoDocument    `bson:"banner_info,omitempty" json:"banner_info,omitempty"`
	Metadata     map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// BannerInfoDocument represents the MongoDB document structure for banner information
type BannerInfoDocument struct {
	RawBanner  string                 `bson:"raw_banner" json:"raw_banner"`
	Service    string                 `bson:"service" json:"service"`
	Protocol   string                 `bson:"protocol" json:"protocol"`
	Version    string                 `bson:"version,omitempty" json:"version,omitempty"`
	Confidence string                 `bson:"confidence" json:"confidence"`
	Metadata   map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// EnrichmentDocument represents the MongoDB document structure for enrichment data
type EnrichmentDocument struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	IP        string                 `bson:"ip" json:"ip"`
	IsUp      bool                   `bson:"is_up" json:"is_up"`
	BatchID   string                 `bson:"batch_id" json:"batch_id"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// ServiceAnalysisDocument represents the MongoDB document structure for service analysis
type ServiceAnalysisDocument struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	IP        string                 `bson:"ip" json:"ip"`
	OpenPorts []PortDocument         `bson:"open_ports" json:"open_ports"`
	BatchID   string                 `bson:"batch_id" json:"batch_id"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	Analysis  map[string]interface{} `bson:"analysis,omitempty" json:"analysis,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// NewMongoDBManager creates a new MongoDB manager
func NewMongoDBManager(connectionString, databaseName, collectionName string) (*MongoDBManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set client options
	clientOptions := options.Client().ApplyURI(connectionString)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(databaseName)
	collection := database.Collection(collectionName)

	// Create indexes for better performance
	err = createIndexes(ctx, collection)
	if err != nil {
		log.L().Warn("Failed to create indexes", zap.Error(err))
	}

	log.L().Info("Connected to MongoDB", zap.String("database", databaseName), zap.String("collection", collectionName))

	return &MongoDBManager{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// createIndexes creates necessary indexes for optimal performance
func createIndexes(ctx context.Context, collection *mongo.Collection) error {
	indexes := []mongo.IndexModel{
		{
			Keys: bson.M{
				"ip":         1,
				"created_at": -1,
			},
			Options: options.Index().SetName("ip_created_at_idx"),
		},
		{
			Keys: bson.M{
				"batch_id":   1,
				"created_at": -1,
			},
			Options: options.Index().SetName("batch_id_created_at_idx"),
		},
		{
			Keys: bson.M{
				"status":     1,
				"created_at": -1,
			},
			Options: options.Index().SetName("status_created_at_idx"),
		},
		{
			Keys: bson.M{
				"worker_id":  1,
				"created_at": -1,
			},
			Options: options.Index().SetName("worker_id_created_at_idx"),
		},
		{
			Keys: bson.M{
				"is_up":      1,
				"open_ports": -1,
			},
			Options: options.Index().SetName("is_up_open_ports_idx"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// SaveScanResult saves a scan result to MongoDB
func (m *MongoDBManager) SaveScanResult(result *domain.ScanResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert domain ScanResult to MongoDB document
	doc := m.convertScanResultToDocument(result)

	// Insert the document
	_, err := m.collection.InsertOne(ctx, doc)
	if err != nil {
		log.L().Error("Failed to save scan result", zap.String("event", "save_failed"),
			zap.String("ip", result.IP), zap.Error(err))
		return fmt.Errorf("failed to save scan result: %w", err)
	}

	log.L().Info("Scan result saved to MongoDB", zap.String("event", "save_success"),
		zap.String("ip", result.IP), zap.Int("open_ports", len(result.GetOpenPorts())))

	return nil
}

// SaveScanResultBatch saves multiple scan results in a batch
func (m *MongoDBManager) SaveScanResultBatch(results []*domain.ScanResult) error {
	if len(results) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Convert domain ScanResults to MongoDB documents
	var documents []interface{}
	for _, result := range results {
		doc := m.convertScanResultToDocument(result)
		documents = append(documents, doc)
	}

	// Insert documents in batch
	_, err := m.collection.InsertMany(ctx, documents)
	if err != nil {
		log.L().Error("Failed to save scan results batch", zap.String("event", "batch_save_failed"),
			zap.Int("count", len(results)), zap.Error(err))
		return fmt.Errorf("failed to save scan results batch: %w", err)
	}

	log.L().Info("Scan results batch saved to MongoDB", zap.String("event", "batch_save_success"),
		zap.Int("count", len(results)))

	return nil
}

// GetScanResult retrieves a scan result by IP
func (m *MongoDBManager) GetScanResult(ip string) (*ScanResultDocument, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doc ScanResultDocument
	err := m.collection.FindOne(ctx, bson.M{"ip": ip}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no scan result found for IP: %s", ip)
		}
		return nil, fmt.Errorf("failed to get scan result: %w", err)
	}

	return &doc, nil
}

// GetScanResultsByBatch retrieves all scan results for a batch
func (m *MongoDBManager) GetScanResultsByBatch(batchID string) ([]*ScanResultDocument, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := m.collection.Find(ctx, bson.M{"batch_id": batchID})
	if err != nil {
		return nil, fmt.Errorf("failed to get scan results by batch: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*ScanResultDocument
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode scan results: %w", err)
	}

	return results, nil
}

// GetScanStats retrieves scanning statistics
func (m *MongoDBManager) GetScanStats() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Aggregate pipeline for statistics
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":         nil,
				"total_scans": bson.M{"$sum": 1},
				"successful_scans": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{bson.M{"$eq": []string{"$status", "completed"}}, 1, 0},
					},
				},
				"failed_scans": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{bson.M{"$eq": []string{"$status", "failed"}}, 1, 0},
					},
				},
				"total_open_ports":  bson.M{"$sum": "$open_ports"},
				"avg_scan_duration": bson.M{"$avg": "$scan_duration"},
			},
		},
	}

	cursor, err := m.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate scan stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	if len(results) == 0 {
		return map[string]interface{}{
			"total_scans":       0,
			"successful_scans":  0,
			"failed_scans":      0,
			"total_open_ports":  0,
			"avg_scan_duration": 0,
		}, nil
	}

	return results[0], nil
}

// convertScanResultToDocument converts domain ScanResult to MongoDB document
func (m *MongoDBManager) convertScanResultToDocument(result *domain.ScanResult) *ScanResultDocument {
	now := time.Now()

	// Convert ports
	var portDocs []PortDocument
	for _, port := range result.Ports {
		portDoc := PortDocument{
			Number:       port.Number,
			Status:       string(port.Status),
			Service:      port.Service,
			Banner:       port.Banner,
			Version:      port.Version,
			ScanTime:     port.ScanTime,
			ResponseTime: port.ResponseTime,
		}

		// Convert banner info if available
		if port.BannerInfo != nil {
			portDoc.BannerInfo = &BannerInfoDocument{
				RawBanner:  port.BannerInfo.RawBanner,
				Service:    port.BannerInfo.Service,
				Protocol:   port.BannerInfo.Protocol,
				Version:    port.BannerInfo.Version,
				Confidence: port.BannerInfo.Confidence,
				Metadata:   port.BannerInfo.Metadata,
			}
		}

		portDocs = append(portDocs, portDoc)
	}

	openPorts := result.GetOpenPorts()
	scanDuration := result.GetScanDuration()

	return &ScanResultDocument{
		IP:            result.IP,
		IsUp:          result.IsUp,
		PingTime:      result.PingTime,
		ScanStartTime: result.ScanStartTime,
		ScanEndTime:   result.ScanEndTime,
		Status:        string(result.Status),
		Error:         result.Error,
		BatchID:       result.BatchID,
		WorkerID:      "", // Will be set by the caller if needed
		Ports:         portDocs,
		OpenPorts:     len(openPorts),
		TotalPorts:    len(result.Ports),
		ScanDuration:  scanDuration,
		CreatedAt:     now,
		UpdatedAt:     now,
		Metadata: map[string]interface{}{
			"source": "port-scanner",
		},
	}
}

// Close closes the MongoDB connection
func (m *MongoDBManager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	log.L().Info("MongoDB connection closed")
	return nil
}
