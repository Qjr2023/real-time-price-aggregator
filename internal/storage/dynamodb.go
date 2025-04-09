package storage

import (
	"log"

	"real-time-price-aggregator/internal/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Storage interface defines data persistence operations
type Storage interface {
	Save(record PriceRecord) error
	Get(asset string) (*PriceRecord, error) // Add Get method
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
	client *dynamodb.DynamoDB
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
func NewDynamoDBStorage(client *dynamodb.DynamoDB) Storage {
	return &DynamoDBStorage{client: client}
}

// Save saves a price record to DynamoDB
func (s *DynamoDBStorage) Save(record PriceRecord) error {
	item, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("prices"),
		Item:      item,
	}

	_, err = s.client.PutItem(input)
	if err != nil {
		log.Printf("Failed to save record for %s: %v", record.Asset, err)
		return err
	}
	return nil
}

// Get retrieves the latest price record for an asset from DynamoDB
func (s *DynamoDBStorage) Get(asset string) (*PriceRecord, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String("prices"),
		KeyConditionExpression: aws.String("asset = :asset"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":asset": {S: aws.String(asset)},
		},
		ScanIndexForward: aws.Bool(false), // Get the latest record (descending order)
		Limit:            aws.Int64(1),    // Only need the most recent record
	}

	result, err := s.client.Query(input)
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

// ConvertPriceDataToRecord converts a PriceData to a PriceRecord
func ConvertPriceDataToRecord(data *types.PriceData) PriceRecord {
	return PriceRecord{
		Asset:     data.Asset,
		Timestamp: data.Timestamp,
		Price:     data.Price,
		UpdatedAt: data.Timestamp,
	}
}
