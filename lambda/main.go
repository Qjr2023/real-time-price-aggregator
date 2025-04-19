package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"real-time-price-aggregator/internal/api"
	"real-time-price-aggregator/internal/cache"
	"real-time-price-aggregator/internal/fetcher"
	"real-time-price-aggregator/internal/storage"

	"github.com/aws/aws-lambda-go/events"
	awslambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/go-redis/redis/v8"
)

// init function
var handler *api.RefreshHandler

// init initializes the Redis client and DynamoDB client
func init() {
	// initialize Redis client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// initialize DynamoDB client
	dynamoClient := storage.NewDynamoDBClient()

	// abtain exchange URLs from environment variables or use defaults
	exchange1 := os.Getenv("EXCHANGE1_URL")
	if exchange1 == "" {
		exchange1 = "http://exchange1:8081/mock/ticker"
	}

	exchange2 := os.Getenv("EXCHANGE2_URL")
	if exchange2 == "" {
		exchange2 = "http://exchange2:8082/mock/ticker"
	}

	exchange3 := os.Getenv("EXCHANGE3_URL")
	if exchange3 == "" {
		exchange3 = "http://exchange3:8083/mock/ticker"
	}

	// initialize fetcher with exchange URLs
	priceFetcher := fetcher.NewFetcher([]string{
		exchange1,
		exchange2,
		exchange3,
	})
	priceCache := cache.NewRedisCache(redisClient)
	priceStorage := storage.NewDynamoDBStorage(dynamoClient)

	// create a new RefreshHandler instance
	handler = api.NewRefreshHandler(priceFetcher, priceCache, priceStorage)
}

// deal with API Gateway requests
func handleAPIGatewayRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get the asset symbol from the request path
	symbol, ok := request.PathParameters["asset"]
	if !ok || symbol == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"msg": "Asset symbol is required"}`,
		}, nil
	}

	// transform the symbol to lowercase for case-insensitive comparison
	symbolLower := strings.ToLower(symbol)

	// check if the asset is supported
	if !api.IsValidAsset(symbolLower) {
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       `{"msg": "Invalid asset format"}`,
		}, nil
	}

	// refresh the price for the asset
	message, statusCode, err := handler.RefreshPrice(symbolLower)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: statusCode,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       fmt.Sprintf(`{"msg": "%s"}`, message),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"message": "%s"}`, message),
	}, nil
}

// handleScheduledEvent handles CloudWatch scheduled events
func handleScheduledEvent(ctx context.Context, event map[string]interface{}) error {
	// abtain the tier from the event
	tier, _ := event["tier"].(string)

	return handler.RefreshAssetsByTier(ctx, tier)
}

// Lambda function handler
func handleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// try to parse the event as an API Gateway event
	var apiGatewayEvent events.APIGatewayProxyRequest
	if err := json.Unmarshal(event, &apiGatewayEvent); err == nil && apiGatewayEvent.Resource != "" {
		// this is an API Gateway event
		return handleAPIGatewayRequest(ctx, apiGatewayEvent)
	}

	// try to parse the event as a CloudWatch Events event
	var cloudWatchEvent map[string]interface{}
	if err := json.Unmarshal(event, &cloudWatchEvent); err == nil {
		// this is a CloudWatch Events event
		return nil, handleScheduledEvent(ctx, cloudWatchEvent)
	}

	return nil, fmt.Errorf("unknown event type: %s", string(event))
}

func main() {
	awslambda.Start(handleRequest)
}
