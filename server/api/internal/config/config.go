package config

import "os"

type Config struct {
	HTTPAddr                  string
	BaseURL                   string
	AssetBaseURL              string
	AWSRegion                 string
	S3Bucket                  string
	DynamoDBImagesTableName   string
	DynamoDBQuizzesTableName  string
	DynamoDBQuizFeedTableName string
	CropQueueURL              string
}

func Load() Config {
	addr := getenv("HTTP_ADDR", ":8080")
	baseURL := getenv("BASE_URL", "http://localhost:8080")
	assetBaseURL := getenv("ASSET_BASE_URL", baseURL)
	awsRegion := getenv("AWS_REGION", "ap-northeast-1")
	s3Bucket := os.Getenv("S3_BUCKET")
	cropQueueURL := os.Getenv("CROP_QUEUE_URL")
	return Config{
		HTTPAddr:                  addr,
		BaseURL:                   baseURL,
		AssetBaseURL:              assetBaseURL,
		AWSRegion:                 awsRegion,
		S3Bucket:                  s3Bucket,
		DynamoDBImagesTableName:   os.Getenv("DYNAMODB_IMAGES_TABLE_NAME"),
		DynamoDBQuizzesTableName:  os.Getenv("DYNAMODB_QUIZZES_TABLE_NAME"),
		DynamoDBQuizFeedTableName: os.Getenv("DYNAMODB_QUIZ_FEED_TABLE_NAME"),
		CropQueueURL:              cropQueueURL,
	}
}

func (c Config) HasDynamoDBConfig() bool {
	return c.DynamoDBImagesTableName != "" ||
		c.DynamoDBQuizzesTableName != "" ||
		c.DynamoDBQuizFeedTableName != ""
}

func (c Config) HasCompleteDynamoDBConfig() bool {
	return c.DynamoDBImagesTableName != "" &&
		c.DynamoDBQuizzesTableName != "" &&
		c.DynamoDBQuizFeedTableName != ""
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
