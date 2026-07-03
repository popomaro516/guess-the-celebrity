package main

import (
	"log"

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
	presigner := localpresign.NewPresigner(cfg.BaseURL)

	router := app.NewRouter(app.Dependencies{
		UploadService:  upload.NewService(imageRepo, presigner, ids, realClock),
		QuizService:    quiz.NewService(quizRepo, imageRepo, queue, ids, realClock),
		AttemptService: attempt.NewService(attemptRepo, quizRepo, imageRepo, ids, realClock),
		BaseURL:        cfg.BaseURL,
	})

	log.Printf("api listening on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
