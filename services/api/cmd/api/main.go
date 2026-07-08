package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/tomy/guess-the-celebrity/services/api/internal/app"
	"github.com/tomy/guess-the-celebrity/services/api/internal/config"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/attempt"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/job"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/upload"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/clock"
	platformdynamodb "github.com/tomy/guess-the-celebrity/services/api/internal/platform/dynamodb"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/idgen"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localdb"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localpresign"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localqueue"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localstorage"
	platforms3 "github.com/tomy/guess-the-celebrity/services/api/internal/platform/s3"
	platformsqs "github.com/tomy/guess-the-celebrity/services/api/internal/platform/sqs"
)

func main() {
	cfg := config.Load()
	gin.SetMode(gin.ReleaseMode)

	ctx := context.Background()
	awsCfg := loadAWSConfig(ctx, cfg)
	store := localdb.NewStore()

	var imageRepo upload.ImageRepository
	var quizRepo quiz.Repository
	var publicFeedRepo quiz.PublicFeedRepository
	var attemptRepo attempt.Repository
	if cfg.DynamoDBTableName == "" {
		localQuizRepo := localdb.NewQuizRepository(store)
		imageRepo = localdb.NewImageRepository(store)
		quizRepo = localQuizRepo
		publicFeedRepo = localQuizRepo
		attemptRepo = localdb.NewAttemptRepository(store)
	} else {
		dynamoClient := awsdynamodb.NewFromConfig(awsCfg)
		dynamoQuizRepo := platformdynamodb.NewQuizRepository(dynamoClient, cfg.DynamoDBTableName)
		imageRepo = platformdynamodb.NewImageRepository(dynamoClient, cfg.DynamoDBTableName)
		quizRepo = dynamoQuizRepo
		publicFeedRepo = dynamoQuizRepo
		attemptRepo = platformdynamodb.NewAttemptRepository(dynamoClient, cfg.DynamoDBTableName)
	}

	ids := idgen.New()
	realClock := clock.New()
	queue := cropQueue(cfg, awsCfg)
	presigner, objects := uploadDependencies(cfg, awsCfg)

	router := app.NewRouter(app.Dependencies{
		UploadService:  upload.NewService(imageRepo, presigner, objects, ids, realClock),
		QuizService:    quiz.NewService(quizRepo, publicFeedRepo, imageRepo, queue, ids, realClock),
		AttemptService: attempt.NewService(attemptRepo, quizRepo, imageRepo, ids, realClock),
		BaseURL:        cfg.BaseURL,
		AssetBaseURL:   cfg.AssetBaseURL,
	})

	log.Printf("api listening on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}

func loadAWSConfig(ctx context.Context, cfg config.Config) aws.Config {
	if cfg.S3Bucket == "" && cfg.DynamoDBTableName == "" && cfg.CropQueueURL == "" {
		return aws.Config{}
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}
	return awsCfg
}

func cropQueue(cfg config.Config, awsCfg aws.Config) job.CropJobQueue {
	if cfg.CropQueueURL == "" {
		return localqueue.NewCropJobQueue()
	}
	return platformsqs.NewCropJobQueue(awssqs.NewFromConfig(awsCfg), cfg.CropQueueURL)
}

func uploadDependencies(cfg config.Config, awsCfg aws.Config) (upload.Presigner, upload.ObjectStore) {
	if cfg.S3Bucket == "" {
		return localpresign.NewPresigner(cfg.BaseURL), localstorage.NewObjectStore()
	}

	client := awss3.NewFromConfig(awsCfg)
	return platforms3.NewPresigner(client, cfg.S3Bucket), platforms3.NewObjectStore(client, cfg.S3Bucket)
}
