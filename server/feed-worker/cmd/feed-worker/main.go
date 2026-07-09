package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/tomy/guess-the-celebrity/server/feed-worker/internal/feed"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	ctx := context.Background()
	config, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("load AWS config failed", "error", err)
		os.Exit(1)
	}
	worker, err := feed.New(
		dynamodb.NewFromConfig(config),
		feed.Config{
			QuizzesTableName: os.Getenv("DYNAMODB_QUIZZES_TABLE_NAME"),
			FeedTableName:    os.Getenv("DYNAMODB_QUIZ_FEED_TABLE_NAME"),
			Logger:           logger,
		},
		time.Now,
	)
	if err != nil {
		logger.Error("configure feed worker failed", "error", err)
		os.Exit(1)
	}
	logger.Info("feed worker starting",
		"aws_region", config.Region,
		"quizzes_table_configured", os.Getenv("DYNAMODB_QUIZZES_TABLE_NAME") != "",
		"feed_table_configured", os.Getenv("DYNAMODB_QUIZ_FEED_TABLE_NAME") != "",
	)

	lambda.Start(func(ctx context.Context) (feed.Result, error) {
		invocationLogger := logger
		if lambdaContext, ok := lambdacontext.FromContext(ctx); ok {
			invocationLogger = logger.With("aws_request_id", lambdaContext.AwsRequestID)
		}
		start := time.Now()
		invocationLogger.Info("feed worker invocation started")
		result, err := worker.Refresh(ctx)
		if err != nil {
			invocationLogger.Error("feed worker invocation failed",
				"duration_ms", time.Since(start).Milliseconds(),
				"error", err,
			)
			return feed.Result{}, err
		}
		invocationLogger.Info("feed worker invocation completed",
			"duration_ms", time.Since(start).Milliseconds(),
			"quiz_count", result.QuizCount,
		)
		return result, nil
	})
}
