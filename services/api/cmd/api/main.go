package main

import (
	"context"
	"log"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/tomy/guess-the-celebrity/services/api/internal/app"
	"github.com/tomy/guess-the-celebrity/services/api/internal/config"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/attempt"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/quiz"
	"github.com/tomy/guess-the-celebrity/services/api/internal/module/upload"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/clock"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/idgen"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localdb"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localpresign"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localqueue"
	"github.com/tomy/guess-the-celebrity/services/api/internal/platform/localstorage"
	platforms3 "github.com/tomy/guess-the-celebrity/services/api/internal/platform/s3"
)

func main() {
	cfg := config.Load()
	gin.SetMode(gin.ReleaseMode)

	store := localdb.NewStore()
	imageRepo := localdb.NewImageRepository(store)
	quizRepo := localdb.NewQuizRepository(store)
	attemptRepo := localdb.NewAttemptRepository(store)
	ids := idgen.New()
	realClock := clock.New()
	queue := localqueue.NewCropJobQueue()
	presigner, objects := uploadDependencies(context.Background(), cfg)

	router := app.NewRouter(app.Dependencies{
		UploadService:  upload.NewService(imageRepo, presigner, objects, ids, realClock),
		QuizService:    quiz.NewService(quizRepo, imageRepo, queue, ids, realClock),
		AttemptService: attempt.NewService(attemptRepo, quizRepo, imageRepo, ids, realClock),
		BaseURL:        cfg.BaseURL,
	})

	log.Printf("api listening on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}

func uploadDependencies(ctx context.Context, cfg config.Config) (upload.Presigner, upload.ObjectStore) {
	if cfg.S3Bucket == "" {
		return localpresign.NewPresigner(cfg.BaseURL), localstorage.NewObjectStore()
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("load AWS config: %v", err)
	}
	client := awss3.NewFromConfig(awsCfg)
	return platforms3.NewPresigner(client, cfg.S3Bucket), platforms3.NewObjectStore(client, cfg.S3Bucket)
}
