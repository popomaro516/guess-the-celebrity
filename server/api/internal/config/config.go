package config

import (
	"os"
	"strconv"
	"strings"
)

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
	AuthDisabled              bool
	CognitoUserPoolID         string
	CognitoAppClientID        string
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
		AuthDisabled:              boolEnv("AUTH_DISABLED"),
		CognitoUserPoolID:         os.Getenv("COGNITO_USER_POOL_ID"),
		CognitoAppClientID:        os.Getenv("COGNITO_APP_CLIENT_ID"),
	}
}

func (c Config) CognitoIssuer() string {
	return "https://cognito-idp." + c.AWSRegion + ".amazonaws.com/" + c.CognitoUserPoolID
}

func (c Config) HasCompleteCognitoConfig() bool {
	return c.CognitoUserPoolID != "" && c.CognitoAppClientID != ""
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

func boolEnv(key string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key)))
	return err == nil && value
}
