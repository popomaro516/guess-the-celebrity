package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tomy/guess-the-celebrity/server/feed-worker/internal/feed"
)

func main() {
	ctx := context.Background()
	config, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}
	worker, err := feed.New(
		dynamodb.NewFromConfig(config),
		feed.Config{
			QuizzesTableName: os.Getenv("DYNAMODB_QUIZZES_TABLE_NAME"),
			FeedTableName:    os.Getenv("DYNAMODB_QUIZ_FEED_TABLE_NAME"),
		},
		time.Now,
	)
	if err != nil {
		log.Fatalf("configure feed worker: %v", err)
	}

	lambda.Start(func(ctx context.Context) (feed.Result, error) {
		return worker.Refresh(ctx)
	})
}
