package storage

import (
	"log"
	"time"

	"real-time-price-aggregator/internal/metrics"
	"real-time-price-aggregator/internal/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Storage interface defines data persistence operations
type Storage interface {
	Save(record PriceRecord) error
	Get(asset string) (*PriceRecord, error)
	BatchGet(assets []string) (map[string]*PriceRecord, error)
}

// PriceRecord represents a price record to be stored in DynamoDB
type PriceRecord struct {
	Asset     string  `dynamodbav:"asset"`
	Timestamp int64   `dynamodbav:"timestamp"`
	Price     float64 `dynamodbav:"price"`
	UpdatedAt int64   `dynamodbav:"updated_at"`
}

// DynamoDBStorage implements the Storage interface
type DynamoDBStorage struct {
	client     *dynamodb.DynamoDB
	sysMetrics *metrics.SystemMetrics
}

// GetClient returns the DynamoDB client
func (s *DynamoDBStorage) GetClient() *dynamodb.DynamoDB {
	return s.client
}

// NewDynamoDBClient creates a new DynamoDB client
func NewDynamoDBClient() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))
	client := dynamodb.New(sess)
	return client
}

// NewDynamoDBStorage creates a new DynamoDB storage instance
func NewDynamoDBStorage(client *dynamodb.DynamoDB, sysMetrics *metrics.SystemMetrics) Storage {
	return &DynamoDBStorage{
		client:     client,
		sysMetrics: sysMetrics,
	}
}

// Save saves a price record to DynamoDB
func (s *DynamoDBStorage) Save(record PriceRecord) error {
	startTime := time.Now()

	item, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName:              aws.String("prices"),
		Item:                   item,
		ReturnConsumedCapacity: aws.String("TOTAL"), // ensure we get consumed capacity
	}

	result, err := s.client.PutItem(input)

	// record metrics
	if s.sysMetrics != nil {
		duration := time.Since(startTime)
		s.sysMetrics.RecordDynamoDBWriteLatency(duration)

		// extract actual consumed capacity units from result
		if result.ConsumedCapacity != nil {
			s.sysMetrics.RecordDynamoDBWriteUnits(*result.ConsumedCapacity.CapacityUnits)
		} else {
			s.sysMetrics.RecordDynamoDBWriteUnits(1.0) // fallback value
		}

		if err != nil {
			s.sysMetrics.RecordDynamoDBError()
		}
	}

	if err != nil {
		log.Printf("Failed to save record for %s: %v", record.Asset, err)
		return err
	}

	return nil
}

// Get retrieves the latest price record for an asset from DynamoDB
func (s *DynamoDBStorage) Get(asset string) (*PriceRecord, error) {
	startTime := time.Now()

	input := &dynamodb.QueryInput{
		TableName:              aws.String("prices"),
		KeyConditionExpression: aws.String("asset = :asset"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":asset": {S: aws.String(asset)},
		},
		ScanIndexForward:       aws.Bool(false),
		Limit:                  aws.Int64(1),
		ReturnConsumedCapacity: aws.String("TOTAL"), // ensure we get consumed capacity
	}

	result, err := s.client.Query(input)

	// record metrics
	if s.sysMetrics != nil {
		duration := time.Since(startTime)
		s.sysMetrics.RecordDynamoDBReadLatency(duration)

		// extract actual consumed capacity units from result
		if result.ConsumedCapacity != nil {
			s.sysMetrics.RecordDynamoDBReadUnits(*result.ConsumedCapacity.CapacityUnits)
		} else {
			s.sysMetrics.RecordDynamoDBReadUnits(0.5) // fallback value
		}

		if err != nil {
			s.sysMetrics.RecordDynamoDBError()
		}
	}

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var record PriceRecord
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// dynamodb.go 修改
// 添加批量获取方法
func (s *DynamoDBStorage) BatchGet(assets []string) (map[string]*PriceRecord, error) {
	startTime := time.Now()

	// 构造BatchGetItem请求
	keys := make([]map[string]*dynamodb.AttributeValue, 0, len(assets))
	for _, asset := range assets {
		keys = append(keys, map[string]*dynamodb.AttributeValue{
			"asset": {S: aws.String(asset)},
		})
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			"prices": {
				Keys: keys,
			},
		},
	}

	result, err := s.client.BatchGetItem(input)

	// 计算指标
	if s.sysMetrics != nil {
		duration := time.Since(startTime)
		s.sysMetrics.RecordDynamoDBReadLatency(duration)
		s.sysMetrics.RecordDynamoDBReadUnits(float64(len(assets)) * 0.5)
		if err != nil {
			s.sysMetrics.RecordDynamoDBError()
		}
	}

	if err != nil {
		return nil, err
	}

	// 处理结果
	records := make(map[string]*PriceRecord)
	if items, ok := result.Responses["prices"]; ok {
		for _, item := range items {
			var record PriceRecord
			if err := dynamodbattribute.UnmarshalMap(item, &record); err != nil {
				continue
			}
			records[record.Asset] = &record
		}
	}

	return records, nil
}

// ConvertPriceDataToRecord converts a PriceData to a PriceRecord
func ConvertPriceDataToRecord(data *types.PriceData) PriceRecord {
	return PriceRecord{
		Asset:     data.Asset,
		Timestamp: data.Timestamp,
		Price:     data.Price,
		UpdatedAt: data.Timestamp,
	}
}
