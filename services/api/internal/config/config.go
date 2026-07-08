package config

import "os"

type Config struct {
	HTTPAddr  string
	BaseURL   string
	AWSRegion string
	S3Bucket  string
}

func Load() Config {
	addr := getenv("HTTP_ADDR", ":8080")
	baseURL := getenv("BASE_URL", "http://localhost:8080")
	awsRegion := getenv("AWS_REGION", "ap-northeast-1")
	s3Bucket := os.Getenv("S3_BUCKET")
	return Config{HTTPAddr: addr, BaseURL: baseURL, AWSRegion: awsRegion, S3Bucket: s3Bucket}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
