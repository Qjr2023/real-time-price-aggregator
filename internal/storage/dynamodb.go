package storage

import (
	// "fmt"
	"log"
	// "time"

	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Storage interface defines data persistence operations
type Storage interface {
	Save(record PriceRecord) error
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

// GetClient 返回 DynamoDB 客户端
func (s *DynamoDBStorage) GetClient() *dynamodb.DynamoDB {
	return s.client
}

// NewDynamoDBClient creates a new DynamoDB client
func NewDynamoDBClient() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("us-west-2"), // 修正为 us-west-2
		LogLevel: aws.LogLevel(aws.LogDebug),
	}))
	client := dynamodb.New(sess)
	log.Printf("DynamoDB client initialized for region: %s", *sess.Config.Region)
	return client
}

// NewDynamoDBStorage creates a new DynamoDB storage instance
func NewDynamoDBStorage(client *dynamodb.DynamoDB) Storage {
	store := &DynamoDBStorage{client: client}
	return store
}

// Save saves a price record to DynamoDB
func (s *DynamoDBStorage) Save(record PriceRecord) error {
	log.Printf("TRANSACTION START - Saving record for %s: %+v", record.Asset, record)
	defer log.Printf("TRANSACTION END - Saving record for %s", record.Asset)

	item, err := dynamodbattribute.MarshalMap(record)
	if err != nil {
		log.Printf("Error marshaling record for %s: %v", record.Asset, err)
		return err
	}

	tableName := "prices"
	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	}

	log.Printf("Sending PutItem to DynamoDB, Table: %s, Item: %+v", tableName, item)
	result, err := s.client.PutItem(input)
	if err != nil {
		log.Printf("Failed to save record for %s to DynamoDB: %v", record.Asset, err)
		return err
	}
	log.Printf("Successfully saved record for %s. DynamoDB response: %+v", record.Asset, result)
	return nil
}
